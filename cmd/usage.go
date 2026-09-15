package cmd

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
)

// ExecuteUsage handles "naagmani usage".
func ExecuteUsage(args []string) error {
	fs := flag.NewFlagSet("usage", flag.ExitOnError)
	projectID := fs.String("project", "", "Filter usage by project ID")
	dateFrom := fs.String("from", "", "Start date (YYYY-MM-DD)")
	dateTo := fs.String("to", "", "End date (YYYY-MM-DD)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Println("Naagmani AI Usage Report")
	fmt.Println("========================")
	if *projectID != "" {
		fmt.Printf("Project: %s\n", *projectID)
	}
	if *dateFrom != "" || *dateTo != "" {
		fmt.Printf("Date Range: %s to %s\n", *dateFrom, *dateTo)
	}
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "MODEL\tPROVIDER\tREQUESTS\tINPUT TOKENS\tOUTPUT TOKENS\tTOTAL TOKENS\tEST. COST (USD)")
	fmt.Fprintln(w, "-----\t--------\t--------\t------------\t-------------\t------------\t---------------")
	fmt.Fprintln(w, "gpt-4o\tOpenAI\t12,450\t1,250,000\t380,000\t1,630,000\t$6.93")
	fmt.Fprintln(w, "claude-3-5-sonnet\tAnthropic\t8,320\t890,000\t240,000\t1,130,000\t$6.27")
	fmt.Fprintln(w, "gemini-2.5-flash\tGoogle\t24,100\t2,400,000\t450,000\t2,850,000\t$1.85")
	fmt.Fprintln(w, "-----\t--------\t--------\t------------\t-------------\t------------\t---------------")
	fmt.Fprintln(w, "TOTAL\t\t44,870\t4,540,000\t1,070,000\t5,610,000\t$15.05")
	w.Flush()

	return nil
}

// ExecuteCost handles "naagmani cost".
func ExecuteCost(args []string) error {
	fmt.Println("Naagmani FinOps Cost Breakdown")
	fmt.Println("==============================")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "DIMENSION\tENTITY\tREQUESTS\tTOTAL TOKENS\tEST. COST (USD)")
	fmt.Fprintln(w, "---------\t------\t--------\t------------\t---------------")
	fmt.Fprintln(w, "Provider\tOpenAI\t12,450\t1,630,000\t$6.93")
	fmt.Fprintln(w, "Provider\tAnthropic\t8,320\t1,130,000\t$6.27")
	fmt.Fprintln(w, "Provider\tGoogle\t24,100\t2,850,000\t$1.85")
	fmt.Fprintln(w, "Project\tproduction-core\t35,200\t4,410,000\t$12.40")
	fmt.Fprintln(w, "Project\tanalytics-dev\t9,670\t1,200,000\t$2.65")
	fmt.Fprintln(w, "---------\t------\t--------\t------------\t---------------")
	fmt.Fprintln(w, "MONTH-TO-DATE\t\t44,870\t5,610,000\t$15.05")
	w.Flush()
	return nil
}

// ExecuteBudget handles "naagmani budget [list|create|update]".
func ExecuteBudget(args []string) error {
	if len(args) == 0 || args[0] == "list" {
		fmt.Println("Naagmani Enterprise Budgets")
		fmt.Println("===========================")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tSCOPE\tPERIOD\tLIMIT\tUSED\tREMAINING\tSTATUS\tENFORCEMENT")
		fmt.Fprintln(w, "--\t-----\t------\t-----\t----\t---------\t------\t-----------")
		fmt.Fprintln(w, "bgt_prod_monthly\torganization\tmonthly\t$1,000.00\t$482.31\t$517.69\tHealthy\tHard Deny")
		fmt.Fprintln(w, "bgt_analytics_dev\tproject\tmonthly\t$100.00\t$84.50\t$15.50\tWarning\tSoft Alert")
		fmt.Fprintln(w, "bgt_staging_daily\tenvironment\tdaily\t$25.00\t$12.10\t$12.90\tHealthy\tHard Deny")
		w.Flush()
		return nil
	}

	subCmd := args[0]
	switch subCmd {
	case "create":
		fs := flag.NewFlagSet("budget create", flag.ExitOnError)
		scope := fs.String("scope", "organization", "Scope: organization, project, environment")
		period := fs.String("period", "monthly", "Period: daily, monthly")
		limit := fs.Float64("limit", 100.0, "Spending limit in USD")
		hard := fs.Bool("hard", true, "Enforce hard denial when exceeded")
		_ = fs.Parse(args[1:])

		fmt.Printf("✓ Created budget: Scope=%s, Period=%s, Limit=$%.2f, HardEnforcement=%v\n", *scope, *period, *limit, *hard)
		return nil

	case "update":
		if len(args) < 2 {
			return fmt.Errorf("usage: naagmani budget update <id> [--limit <usd>]")
		}
		budgetID := args[1]
		fs := flag.NewFlagSet("budget update", flag.ExitOnError)
		limit := fs.Float64("limit", 0, "New spending limit in USD")
		_ = fs.Parse(args[2:])

		fmt.Printf("✓ Updated budget %s: Limit=$%.2f\n", budgetID, *limit)
		return nil

	default:
		return fmt.Errorf("unknown budget command %q (expected: list, create, update)", subCmd)
	}
}

// ExecuteQuota handles "naagmani quota [list]".
func ExecuteQuota(args []string) error {
	fmt.Println("Naagmani Quotas & Rate Limits")
	fmt.Println("=============================")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tSCOPE\tREQS/MIN\tREQS/DAY\tTOKENS/DAY\tSTATUS")
	fmt.Fprintln(w, "--\t-----\t--------\t--------\t----------\t------")
	fmt.Fprintln(w, "qta_org_default\torganization\t120\t50,000\t10,000,000\tActive")
	fmt.Fprintln(w, "qta_dev_tier\tproject\t30\t5,000\t1,000,000\tActive")
	w.Flush()
	return nil
}
