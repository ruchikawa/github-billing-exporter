// https://docs.github.com/en/free-pro-team@latest/rest/reference/billing
package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type apiMode int

const (
	orgMode apiMode = iota + 1
	userMode
)

var (
	totalMinutesUsedGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "total_minutes_used",
			Help: "github actions total minutes used",
		},
		[]string{"owner"},
	)
	totalPaidMinutesUsedGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "total_paid_minutes_used",
			Help: "github actions total paid minutes used",
		},
		[]string{"owner"},
	)
	includedMinutesGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "included_minutes",
			Help: "github actions included minutes",
		},
		[]string{"owner"},
	)
	minutesUsedBreakdownGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "minutes_used_breakdown",
			Help: "github actions minutes used breakdown",
		},
		[]string{"owner", "os"},
	)

	totalGigabytesBandwidthUsedGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "total_gigabytes_bandwidth_used",
			Help: "github packages included minutes",
		},
		[]string{"owner"},
	)
	totalPaidGigabytesBandwidthUsedGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "total_paid_gigabytes_bandwidth_used",
			Help: "github packages total paid gigabytes bandwidth used",
		},
		[]string{"owner"},
	)
	includedGigabytesBandwidthGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "included_gigabytes_bandwidth",
			Help: "github packages included gigabytes bandwidth",
		},
		[]string{"owner"},
	)

	daysLeftInBillingCycleGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "days_left_in_billing_cycle",
			Help: "github shared storage days left in billing cycle",
		},
		[]string{"owner"},
	)
	estimatedPaidStorageForMonthGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "estimated_paid_storage_for_month",
			Help: "github shared storage estimated paid storage for month",
		},
		[]string{"owner"},
	)
	estimatedStorageForMonthGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "estimated_storage_for_month",
			Help: "github shared storage estimated storage for month",
		},
		[]string{"owner"},
	)
)

type actionsBilling struct {
	TotalMinutesUsed     int    `json:"total_minutes_used"`
	TotalPaidMinutesUsed string `json:"total_paid_minutes_used"`
	IncludedMinutes      int    `json:"included_minutes"`
	MinutesUsedBreakdown struct {
		UBUNTU  int `json:"UBUNTU"`
		MACOS   int `json:"MACOS"`
		WINDOWS int `json:"WINDOWS"`
	} `json:"minutes_used_breakdown"`
}

type packagesBilling struct {
	TotalGigabytesBandwidthUsed     int `json:"total_gigabytes_bandwidth_used"`
	TotalPaidGigabytesBandwidthUsed int `json:"total_paid_gigabytes_bandwidth_used"`
	IncludedGigabytesBandwidth      int `json:"included_gigabytes_bandwidth"`
}

type sharedStorageBilling struct {
	DaysLeftInBillingCycle       int `json:"days_left_in_billing_cycle"`
	EstimatedPaidStorageForMonth int `json:"estimated_paid_storage_for_month"`
	EstimatedStorageForMonth     int `json:"estimated_storage_for_month"`
}

func init() {
	prometheus.MustRegister(totalMinutesUsedGauge)
	prometheus.MustRegister(totalPaidMinutesUsedGauge)
	prometheus.MustRegister(includedMinutesGauge)
	prometheus.MustRegister(minutesUsedBreakdownGauge)

	prometheus.MustRegister(totalGigabytesBandwidthUsedGauge)
	prometheus.MustRegister(totalPaidGigabytesBandwidthUsedGauge)
	prometheus.MustRegister(includedGigabytesBandwidthGauge)

	prometheus.MustRegister(daysLeftInBillingCycleGauge)
	prometheus.MustRegister(estimatedPaidStorageForMonthGauge)
	prometheus.MustRegister(estimatedStorageForMonthGauge)
}

func getGitHubActionsBilling(mode apiMode, args *Args) {
	var (
		client  = &http.Client{}
		baseURL string
		owner   string
	)

	switch mode {
	case orgMode:
		baseURL = fmt.Sprintf("https://api.github.com/orgs/%s/settings/billing/actions", args.Organization)
		owner = args.Organization
	case userMode:
		baseURL = fmt.Sprintf("https://api.github.com/users/%s/settings/billing/actions", args.User)
		owner = args.User
	default:
		log.Fatal("Invalid select mode")
	}

	for {
		var p actionsBilling
		req, err := http.NewRequest("GET", baseURL, nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("token %s", args.Token))

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		err = json.NewDecoder(resp.Body).Decode(&p)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()

		f, err := strconv.ParseFloat(p.TotalPaidMinutesUsed, 64)
		if err != nil {
			log.Fatal(err)
		}

		totalMinutesUsedGauge.WithLabelValues(owner).Set(float64(p.TotalMinutesUsed))
		totalPaidMinutesUsedGauge.WithLabelValues(owner).Set(f)
		includedMinutesGauge.WithLabelValues(owner).Set(float64(p.IncludedMinutes))
		minutesUsedBreakdownGauge.WithLabelValues(owner, "ubuntu").Set(float64(p.MinutesUsedBreakdown.UBUNTU))
		minutesUsedBreakdownGauge.WithLabelValues(owner, "macos").Set(float64(p.MinutesUsedBreakdown.MACOS))
		minutesUsedBreakdownGauge.WithLabelValues(owner, "windows").Set(float64(p.MinutesUsedBreakdown.WINDOWS))

		time.Sleep(time.Duration(args.Refresh) * time.Second)
	}
}

func getGitHubPackagesBilling(mode apiMode, args *Args) {
	var (
		client  = &http.Client{}
		baseURL string
		owner   string
	)

	switch mode {
	case orgMode:
		baseURL = fmt.Sprintf("https://api.github.com/orgs/%s/settings/billing/packages", args.Organization)
		owner = args.Organization
	case userMode:
		baseURL = fmt.Sprintf("https://api.github.com/users/%s/settings/billing/packages", args.User)
		owner = args.User
	default:
		log.Fatal("Invalid select mode")
	}

	for {
		var p packagesBilling
		req, err := http.NewRequest("GET", baseURL, nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("token %s", args.Token))

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		err = json.NewDecoder(resp.Body).Decode(&p)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()

		totalGigabytesBandwidthUsedGauge.WithLabelValues(owner).Set(float64(p.TotalGigabytesBandwidthUsed))
		totalPaidGigabytesBandwidthUsedGauge.WithLabelValues(owner).Set(float64(p.TotalPaidGigabytesBandwidthUsed))
		includedGigabytesBandwidthGauge.WithLabelValues(owner).Set(float64(p.IncludedGigabytesBandwidth))

		time.Sleep(time.Duration(args.Refresh) * time.Second)
	}
}

func getGitHubSharedStorageBilling(mode apiMode, args *Args) {
	var (
		client  = &http.Client{}
		baseURL string
		owner   string
	)

	switch mode {
	case orgMode:
		baseURL = fmt.Sprintf("https://api.github.com/orgs/%s/settings/billing/shared-storage", args.Organization)
		owner = args.Organization
	case userMode:
		baseURL = fmt.Sprintf("https://api.github.com/users/%s/settings/billing/shared-storage", args.User)
		owner = args.User
	default:
		log.Fatal("Invalid select mode")
	}

	for {
		var p sharedStorageBilling
		req, err := http.NewRequest("GET", baseURL, nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("token %s", args.Token))

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		err = json.NewDecoder(resp.Body).Decode(&p)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()

		daysLeftInBillingCycleGauge.WithLabelValues(owner).Set(float64(p.DaysLeftInBillingCycle))
		estimatedPaidStorageForMonthGauge.WithLabelValues(owner).Set(float64(p.EstimatedPaidStorageForMonth))
		estimatedStorageForMonthGauge.WithLabelValues(owner).Set(float64(p.EstimatedStorageForMonth))

		time.Sleep(time.Duration(args.Refresh) * time.Second)
	}
}
