package wallet

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

type HttpClient interface {
	Get(string) (*http.Response, error)
}

type feeCache struct {
	fees        *Fees
	lastUpdated time.Time
}

type Fees struct {
	Priority int64 `json:"priority"`
	Normal   int64 `json:"normal"`
	Economic int64 `json:"economic"`
}

type FeeProvider struct {
	MaxFee      int64
	PriorityFee int64
	NormalFee   int64
	EconomicFee int64
	FeeAPI      string

	HttpClient HttpClient

	cache *feeCache
}

func NewFeeProvider(maxFee, priorityFee, normalFee, economicFee int64, feeAPI string, proxy proxy.Dialer) *FeeProvider {
	fp := FeeProvider{
		MaxFee:      maxFee,
		PriorityFee: priorityFee,
		NormalFee:   normalFee,
		EconomicFee: economicFee,
		FeeAPI:      feeAPI,
		cache:       new(feeCache),
	}
	dial := net.Dial
	if proxy != nil {
		dial = proxy.Dial
	}
	tbTransport := &http.Transport{Dial: dial}
	httpClient := &http.Client{Transport: tbTransport, Timeout: time.Second * 10}
	fp.HttpClient = httpClient
	return &fp
}

func (fp *FeeProvider) GetFeePerByte(feeLevel FeeLevel) int64 {
	if fp.FeeAPI == "" {
		return fp.defaultFee(feeLevel)
	}
	fees := new(Fees)
	if time.Since(fp.cache.lastUpdated) > time.Minute {
		resp, err := fp.HttpClient.Get(fp.FeeAPI)
		if err != nil {
			return fp.defaultFee(feeLevel)
		}

		defer resp.Body.Close()

		err = json.NewDecoder(resp.Body).Decode(&fees)
		if err != nil {
			return fp.defaultFee(feeLevel)
		}
		fp.cache.lastUpdated = time.Now()
		fp.cache.fees = fees
	} else {
		fees = fp.cache.fees
	}
	switch feeLevel {
	case PRIORITY:
		return fp.selectFee(fees.Priority, PRIORITY)
	case NORMAL:
		return fp.selectFee(fees.Normal, PRIORITY)
	case ECONOMIC:
		return fp.selectFee(fees.Economic, PRIORITY)
	case FEE_BUMP:
		return fp.selectFee(fees.Priority, PRIORITY)
	default:
		return fp.NormalFee
	}
}

func (fp *FeeProvider) selectFee(fee int64, feeLevel FeeLevel) int64 {
	if fee > fp.MaxFee {
		return fp.MaxFee
	} else if fee == 0 {
		return fp.defaultFee(feeLevel)
	} else {
		return fee
	}
}

func (fp *FeeProvider) defaultFee(feeLevel FeeLevel) int64 {
	switch feeLevel {
	case PRIORITY:
		return fp.PriorityFee
	case NORMAL:
		return fp.NormalFee
	case ECONOMIC:
		return fp.EconomicFee
	case FEE_BUMP:
		return fp.PriorityFee
	default:
		return fp.NormalFee
	}
}

// December 2023
func DefaultFeeProvider() *FeeProvider {
	return NewFeeProvider(int64(1000), int64(50), int64(30), int64(20), "", nil)
}
