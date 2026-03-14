package dtos

// Balance represents the balance of an account.
type Balance struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			TotalEquity           string `json:"totalEquity"` // real with PnL and borrow in USD
			TotalMarginBalance    string `json:"totalMarginBalance"`
			AccountType           string `json:"accountType"`
			TotalAvailableBalance string `json:"totalAvailableBalance"` // clean, wo PnL and borrow in USD
			Coin                  []struct {
				Equity        string `json:"equity"`
				UsdValue      string `json:"usdValue"`
				UnrealisedPnl string `json:"unrealisedPnl"`
				BorrowAmount  string `json:"borrowAmount"`
				WalletBalance string `json:"walletBalance"`
				Locked        string `json:"locked"`
				Coin          string `json:"coin"`
			} `json:"coin"`
		} `json:"list"`
	} `json:"result"`
	RetExtInfo struct {
	} `json:"retExtInfo"`
	Time int64 `json:"time"`
}
