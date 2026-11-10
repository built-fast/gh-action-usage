package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

// usageItem is a single SKU line from the billing usage summary API.
type usageItem struct {
	Product        string  `json:"product"`
	SKU            string  `json:"sku"`
	UnitType       string  `json:"unitType"`
	PricePerUnit   float64 `json:"pricePerUnit"`
	GrossQuantity  float64 `json:"grossQuantity"`
	GrossAmount    float64 `json:"grossAmount"`
	DiscountAmount float64 `json:"discountAmount"`
	NetQuantity    float64 `json:"netQuantity"`
	NetAmount      float64 `json:"netAmount"`
}

type usageSummary struct {
	UsageItems []usageItem `json:"usageItems"`
}

// lineItem is a single daily usage line from the billing usage API.
type lineItem struct {
	Product        string  `json:"product"`
	UnitType       string  `json:"unitType"`
	Quantity       float64 `json:"quantity"`
	GrossAmount    float64 `json:"grossAmount"`
	DiscountAmount float64 `json:"discountAmount"`
	NetAmount      float64 `json:"netAmount"`
	RepositoryName string  `json:"repositoryName"`
}

// repoUsage is Actions usage aggregated for one repository.
type repoUsage struct {
	Repo           string
	Minutes        float64
	GrossAmount    float64
	DiscountAmount float64
	NetAmount      float64
}

type restClient interface {
	Get(path string, resp interface{}) error
}

// billingBase returns the billing API path prefix for an owner, which differs
// between organizations and users, along with the owner's account type.
func billingBase(client restClient, owner string) (base, ownerType string, err error) {
	var acct struct {
		Type string `json:"type"`
	}
	if err := client.Get("users/"+url.PathEscape(owner), &acct); err != nil {
		return "", "", fmt.Errorf("looking up %s: %w", owner, err)
	}

	switch acct.Type {
	case "Organization":
		return "organizations/" + url.PathEscape(owner) + "/settings/billing", acct.Type, nil
	case "User":
		return "users/" + url.PathEscape(owner) + "/settings/billing", acct.Type, nil
	default:
		return "", "", fmt.Errorf("unsupported account type %q for %s", acct.Type, owner)
	}
}

func periodQuery(p period) url.Values {
	q := url.Values{}
	q.Set("year", strconv.Itoa(p.Year))
	q.Set("month", strconv.Itoa(int(p.Month)))
	return q
}

// fetchSummary returns the Actions usage summary for the target and period.
//
// See https://docs.github.com/rest/billing/usage
func fetchSummary(client restClient, t target, p period) (usageSummary, error) {
	base, ownerType, err := billingBase(client, t.Owner)
	if err != nil {
		return usageSummary{}, err
	}

	q := periodQuery(p)
	q.Set("product", "Actions")
	if t.Repo != "" {
		q.Set("repository", t.String())
	}

	var summary usageSummary
	if err := client.Get(base+"/usage/summary?"+q.Encode(), &summary); err != nil {
		return usageSummary{}, withHint(err, ownerType)
	}
	return summary, nil
}

// fetchByRepo returns Actions usage for an owner aggregated per repository,
// sorted by net amount, then gross amount, descending.
//
// See https://docs.github.com/rest/billing/usage
func fetchByRepo(client restClient, owner string, p period) ([]repoUsage, error) {
	base, ownerType, err := billingBase(client, owner)
	if err != nil {
		return nil, err
	}

	q := periodQuery(p)
	q.Set("product", "actions")

	var resp struct {
		UsageItems []lineItem `json:"usageItems"`
	}
	if err := client.Get(base+"/usage?"+q.Encode(), &resp); err != nil {
		return nil, withHint(err, ownerType)
	}

	return aggregateByRepo(resp.UsageItems), nil
}

const noRepo = "(no repository)"

func aggregateByRepo(items []lineItem) []repoUsage {
	byRepo := map[string]*repoUsage{}
	for _, it := range items {
		if !strings.EqualFold(it.Product, "actions") {
			continue
		}

		name := it.RepositoryName
		if name == "" {
			name = noRepo
		}
		u, ok := byRepo[name]
		if !ok {
			u = &repoUsage{Repo: name}
			byRepo[name] = u
		}

		if strings.EqualFold(it.UnitType, "minutes") {
			u.Minutes += it.Quantity
		}
		u.GrossAmount += it.GrossAmount
		u.DiscountAmount += it.DiscountAmount
		u.NetAmount += it.NetAmount
	}

	out := make([]repoUsage, 0, len(byRepo))
	for _, u := range byRepo {
		out = append(out, *u)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].NetAmount != out[j].NetAmount {
			return out[i].NetAmount > out[j].NetAmount
		}
		if out[i].GrossAmount != out[j].GrossAmount {
			return out[i].GrossAmount > out[j].GrossAmount
		}
		return out[i].Repo < out[j].Repo
	})
	return out
}

func withHint(err error, ownerType string) error {
	var httpErr *api.HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusForbidden && ownerType == "Organization" {
		return fmt.Errorf("%w\nhint: organization billing requires the admin:org scope; run: gh auth refresh -s admin:org", err)
	}
	return err
}
