package main

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"text/tabwriter"
)

func render(w io.Writer, t target, p period, s usageSummary) error {
	if _, err := fmt.Fprintf(w, "GitHub Actions usage for %s (%s)\n\n", t, p); err != nil {
		return err
	}

	if len(s.UsageItems) == 0 {
		_, err := fmt.Fprintln(w, "No usage.")
		return err
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tw, "SKU\tQuantity\tUnit\tGross\tDiscount\tNet\t")

	var gross, discount, net float64
	for _, it := range s.UsageItems {
		gross += it.GrossAmount
		discount += it.DiscountAmount
		net += it.NetAmount
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t\n",
			it.SKU, formatQuantity(it.GrossQuantity), it.UnitType,
			formatMoney(it.GrossAmount), formatMoney(it.DiscountAmount), formatMoney(it.NetAmount))
	}
	fmt.Fprintf(tw, "Total\t\t\t%s\t%s\t%s\t\n", formatMoney(gross), formatMoney(discount), formatMoney(net))

	return tw.Flush()
}

func formatMoney(v float64) string {
	r := roundTo(v, 2)
	sign := ""
	if r < 0 {
		sign = "-"
	}
	return sign + "$" + addCommas(strconv.FormatFloat(math.Abs(r), 'f', 2, 64))
}

func formatQuantity(v float64) string {
	if v == math.Trunc(v) {
		return addCommas(strconv.FormatFloat(v, 'f', 0, 64))
	}
	return addCommas(strconv.FormatFloat(roundTo(v, 2), 'f', 2, 64))
}

func roundTo(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}

// addCommas inserts thousands separators into a non-negative decimal string.
func addCommas(s string) string {
	intPart, frac, hasFrac := strings.Cut(s, ".")
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if hasFrac {
		b.WriteByte('.')
		b.WriteString(frac)
	}
	return b.String()
}

func renderByRepo(w io.Writer, t target, p period, usage []repoUsage) error {
	if _, err := fmt.Fprintf(w, "GitHub Actions usage for %s by repository (%s)\n\n", t, p); err != nil {
		return err
	}

	if len(usage) == 0 {
		_, err := fmt.Fprintln(w, "No usage.")
		return err
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tw, "Repository\tMinutes\tGross\tDiscount\tNet\t")

	var total repoUsage
	for _, u := range usage {
		total.Minutes += u.Minutes
		total.GrossAmount += u.GrossAmount
		total.DiscountAmount += u.DiscountAmount
		total.NetAmount += u.NetAmount
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t\n", u.Repo, formatQuantity(u.Minutes),
			formatMoney(u.GrossAmount), formatMoney(u.DiscountAmount), formatMoney(u.NetAmount))
	}
	fmt.Fprintf(tw, "Total\t%s\t%s\t%s\t%s\t\n", formatQuantity(total.Minutes),
		formatMoney(total.GrossAmount), formatMoney(total.DiscountAmount), formatMoney(total.NetAmount))

	return tw.Flush()
}
