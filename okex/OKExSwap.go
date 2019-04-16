package okex

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	. "github.com/wudian/GoEx"
	"github.com/pkg/errors"
	"log"
	"strings"
	"time"
)

const (
	/*
	  http headers
	*/
	OK_ACCESS_KEY        = "OK-ACCESS-KEY"
	OK_ACCESS_SIGN       = "OK-ACCESS-SIGN"
	OK_ACCESS_TIMESTAMP  = "OK-ACCESS-TIMESTAMP"
	OK_ACCESS_PASSPHRASE = "OK-ACCESS-PASSPHRASE"

	/**
	  paging params
	*/
	OK_FROM  = "OK-FROM"
	OK_TO    = "OK-TO"
	OK_LIMIT = "OK-LIMIT"

	CONTENT_TYPE = "Content-Type"
	ACCEPT       = "Accept"
	COOKIE       = "Cookie"
	LOCALE       = "locale="

	APPLICATION_JSON      = "application/json"
	APPLICATION_JSON_UTF8 = "application/json; charset=UTF-8"

	/*
	  i18n: internationalization
	*/
	ENGLISH            = "en_US"
	SIMPLIFIED_CHINESE = "zh_CN"
	//zh_TW || zh_HK
	TRADITIONAL_CHINESE = "zh_HK"

	/*
	  http methods
	*/
	GET    = "GET"
	POST   = "POST"
	DELETE = "DELETE"

	/*
	 others
	*/
	ResultDataJsonString = "resultDataJsonString"
	ResultPageJsonString = "resultPageJsonString"

	BTC_USD_SWAP = "BTC-USD-SWAP"
	LTC_USD_SWAP = "LTC-USD-SWAP"
	ETH_USD_SWAP = "ETH-USD-SWAP"
	ETC_USD_SWAP = "ETC-USD-SWAP"
	BCH_USD_SWAP = "BCH-USD-SWAP"
	BSV_USD_SWAP = "BSV-USD-SWAP"
	EOS_USD_SWAP = "EOS-USD-SWAP"
	XRP_USD_SWAP = "XRP-USD-SWAP"

	/*Rest Endpoint*/
	Endpoint              = "https://www.okex.com"
	GET_ACCOUNTS          = "/api/swap/v3/accounts"
	PLACE_ORDER           = "/api/swap/v3/order"
	CANCEL_ORDER          = "/api/swap/v3/cancel_order/%s/%s"
	GET_ORDER             = "/api/swap/v3/orders/%s/%s"
	GET_POSITION          = "/api/swap/v3/%s/position"
	GET_DEPTH             = "/api/swap/v3/instruments/%s/depth?size=%d"
	GET_TICKER            = "/api/swap/v3/instruments/%s/ticker"
	GET_UNFINISHED_ORDERS = "/api/swap/v3/orders/%s?status=%d&from=%d&limit=%d"
)

type BaseResponse struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Result       bool   `json:"result,string"`
}

type OKExSwap struct {
	config *APIConfig
}

func NewOKExSwap(config *APIConfig) *OKExSwap {
	return &OKExSwap{config: config}
}

func (ok *OKExSwap) GetExchangeName() string {
	return OKEX_SWAP
}

func (ok *OKExSwap) GetFutureTicker(currencyPair CurrencyPair, contractType string) (*Ticker, error) {
	var resp BaseTickerInfo
	err := ok.doRequest("GET", fmt.Sprintf(GET_TICKER, contractType), "", &resp)
	if err != nil {
		return nil, err
	}

	date, _ := time.Parse(time.RFC3339, resp.Timestamp)
	return &Ticker{
		Pair: currencyPair,
		Last: resp.Last,
		Low:  resp.Low24h,
		High: resp.High24h,
		Vol:  resp.Volume24h,
		Date: uint64(date.UnixNano() / int64(time.Millisecond))}, nil
}

func (ok *OKExSwap) GetFutureDepth(currencyPair CurrencyPair, contractType string, size int) (*Depth, error) {
	var resp SwapInstrumentDepth

	err := ok.doRequest("GET", fmt.Sprintf(GET_DEPTH, contractType, size), "", &resp)
	if err != nil {
		return nil, err
	}

	var dep Depth
	dep.ContractType = contractType
	dep.Pair = currencyPair
	dep.UTime, _ = time.Parse(time.RFC3339, resp.Timestamp)

	for _, v := range resp.Bids {
		dep.BidList = append(dep.BidList, DepthRecord{
			Price:  ToFloat64(v[0]),
			Amount: ToFloat64(v[1])})
	}

	for i := len(resp.Asks) - 1; i >= 0; i-- {
		dep.AskList = append(dep.AskList, DepthRecord{
			Price:  ToFloat64(resp.Asks[i][0]),
			Amount: ToFloat64(resp.Asks[i][1])})
	}

	return &dep, nil
}

func (ok *OKExSwap) GetFutureUserinfo() (*FutureAccount, error) {
	var infos SwapAccounts

	err := ok.doRequest("GET", GET_ACCOUNTS, "", &infos)
	if err != nil {
		return nil, err
	}

	//log.Println(infos)
	acc := FutureAccount{}
	acc.FutureSubAccounts = make(map[Currency]FutureSubAccount, 2)

	for _, account := range infos.Info {
		subAcc := FutureSubAccount{AccountRights: account.Equity,
			KeepDeposit: account.Margin, ProfitReal: account.RealizedPnl, ProfitUnreal: account.UnrealizedPnl, RiskRate: account.MarginRatio}
		switch account.InstrumentId {
		case BTC_USD_SWAP:
			subAcc.Currency = BTC
		case LTC_USD_SWAP:
			subAcc.Currency = LTC
		case ETH_USD_SWAP:
			subAcc.Currency = ETH
		case ETC_USD_SWAP:
			subAcc.Currency = ETC
		case BCH_USD_SWAP:
			subAcc.Currency = BCH
		case BSV_USD_SWAP:
			subAcc.Currency = BSV
		case EOS_USD_SWAP:
			subAcc.Currency = EOS
		case XRP_USD_SWAP:
			subAcc.Currency = XRP
		default:
			subAcc.Currency = UNKNOWN
		}
		acc.FutureSubAccounts[subAcc.Currency] = subAcc
	}

	return &acc, nil
}

/*
 OKEX swap api parameter's definition
 @author Lingting Fu
 @date 2018-12-27
 @version 1.0.0
*/

type BasePlaceOrderInfo struct {
	ClientOid  string `json:"client_oid"`
	Price      string `json:"price"`
	MatchPrice string `json:"match_price"`
	Type       string `json:"type"`
	Size       string `json:"size"`
}

type PlaceOrderInfo struct {
	BasePlaceOrderInfo
	InstrumentId string `json:"instrument_id"`
}

type PlaceOrdersInfo struct {
	InstrumentId string                `json:"instrument_id"`
	OrderData    []*BasePlaceOrderInfo `json:"order_data"`
}

func (ok *OKExSwap) PlaceFutureOrder(currencyPair CurrencyPair, contractType, price, amount string, openType, matchPrice, leverRate int) (string, error) {

	reqBody, _, _ := BuildRequestBody(PlaceOrderInfo{
		BasePlaceOrderInfo{ClientOid: strings.Replace(uuid.New().String(), "-", "", 32), Price: price, MatchPrice: "0", Type: fmt.Sprint(openType), Size: amount},
		contractType,
	})

	var resp struct {
		BaseResponse
		OrderID   string `json:"order_id"`
		ClientOid string `json:"client_oid"`
	}

	err := ok.doRequest("POST", PLACE_ORDER, reqBody, &resp)
	if err != nil {
		return "", err
	}

	if resp.ErrorMessage != "" {
		return "", errors.New(fmt.Sprintf("%s:%s", resp.ErrorCode, resp.ErrorMessage))
	}

	return resp.OrderID, nil
}

func (ok *OKExSwap) FutureCancelOrder(currencyPair CurrencyPair, contractType, orderId string) (bool, error) {
	var cancelParam struct {
		OrderId      string `json:"order_id"`
		InstrumentId string `json:"instrument_id"`
	}

	var resp SwapCancelOrderResult

	cancelParam.InstrumentId = contractType
	cancelParam.OrderId = orderId

	//req, _, _ := BuildRequestBody(cancelParam)

	err := ok.doRequest("POST", fmt.Sprintf(CANCEL_ORDER, contractType, orderId), "", &resp)
	if err != nil {
		return false, err
	}

	return resp.Result, nil
}

func (ok *OKExSwap) parseOrder(ord BaseOrderInfo) FutureOrder {
	oTime, _ := time.Parse(time.RFC3339, ord.Timestamp)
	return FutureOrder{
		OrderID2:   ord.OrderId,
		Amount:     ord.Size,
		Price:      ord.Price,
		DealAmount: ord.FilledQty,
		AvgPrice:   ord.PriceAvg,
		OType:      ord.Type,
		Status:     ok.AdaptTradeStatus(ord.Status),
		Fee:        ord.Fee,
		OrderTime:  oTime.UnixNano() / int64(time.Millisecond)}
}

func (ok *OKExSwap) GetUnfinishFutureOrders(currencyPair CurrencyPair, contractType string) ([]FutureOrder, error) {
	var (
		resp  SwapOrdersInfo
		resp2 SwapOrdersInfo
	)

	err := ok.doRequest("GET", fmt.Sprintf(GET_UNFINISHED_ORDERS, contractType, ORDER_UNFINISH, 1, 100), "", &resp)
	if err != nil {
		return nil, err
	}

	if resp.Message != "" {
		return nil, errors.New(fmt.Sprintf("{\"ErrCode\":%d,\"ErrMessage\":\"%s\"", resp.Code, resp.Message))
	}

	var orders []FutureOrder
	for _, info := range resp.OrderInfo {
		ord := ok.parseOrder(info)
		ord.Currency = currencyPair
		ord.ContractName = contractType
		orders = append(orders, ord)
	}

	err = ok.doRequest("GET", fmt.Sprintf(GET_UNFINISHED_ORDERS, contractType, ORDER_PART_FINISH, 1, 100), "", &resp2)
	if err != nil {
		log.Println(err)
	}

	for _, info := range resp2.OrderInfo {
		ord := ok.parseOrder(info)
		ord.Currency = currencyPair
		ord.ContractName = contractType
		orders = append(orders, ord)
	}

	//log.Println(len(orders))
	return orders, nil
}

/**
 *获取订单信息
 */
func (ok *OKExSwap) GetFutureOrders(orderIds []string, currencyPair CurrencyPair, contractType string) ([]FutureOrder, error) {
	panic("")
}

/**
 *获取单个订单信息
 */
func (ok *OKExSwap) GetFutureOrder(orderId string, currencyPair CurrencyPair, contractType string) (*FutureOrder, error) {
	var getOrderParam struct {
		OrderId      string `json:"order_id"`
		InstrumentId string `json:"instrument_id"`
	}

	var resp struct {
		BizWarmTips
		BaseOrderInfo
	}

	getOrderParam.OrderId = orderId
	getOrderParam.InstrumentId = contractType

	//reqBody, _, _ := BuildRequestBody(getOrderParam)

	err := ok.doRequest("GET", fmt.Sprintf(GET_ORDER, contractType, orderId), "", &resp)
	if err != nil {
		return nil, err
	}

	if resp.Message != "" {
		return nil, errors.New(fmt.Sprintf("{\"ErrCode\":%d,\"ErrMessage\":\"%s\"}", resp.Code, resp.Message))
	}

	oTime, err := time.Parse(time.RFC3339, resp.Timestamp)

	return &FutureOrder{
		Currency:     currencyPair,
		ContractName: contractType,
		OrderID2:     resp.OrderId,
		Amount:       resp.Size,
		Price:        resp.Price,
		DealAmount:   resp.FilledQty,
		AvgPrice:     resp.PriceAvg,
		OType:        resp.Type,
		Fee:          resp.Fee,
		Status:       ok.AdaptTradeStatus(resp.Status),
		OrderTime:    oTime.UnixNano() / int64(time.Millisecond),
	}, nil
}

func (ok *OKExSwap) GetFuturePosition(currencyPair CurrencyPair, contractType string) ([]FuturePosition, error) {
	var resp SwapPosition

	err := ok.doRequest("GET", fmt.Sprintf(GET_POSITION, contractType), "", &resp)
	if err != nil {
		return nil, err
	}

	var positions []FuturePosition

	positions = append(positions, FuturePosition{
		ContractType: contractType,
		Symbol:       currencyPair})

	var (
		buyPosition  SwapPositionHolding
		sellPosition SwapPositionHolding
	)

	if len(resp.Holding) > 0 {
		if resp.Holding[0].Side == "long" {
			buyPosition = resp.Holding[0]
			if len(resp.Holding) == 2 {
				sellPosition = resp.Holding[1]
			}
		} else {
			sellPosition = resp.Holding[0]
			if len(resp.Holding) == 2 {
				buyPosition = resp.Holding[1]
			}
		}

		positions[0].ForceLiquPrice = buyPosition.LiquidationPrice
		positions[0].BuyAmount = buyPosition.Position
		positions[0].BuyAvailable = buyPosition.AvailPosition
		positions[0].BuyPriceAvg = buyPosition.AvgCost
		positions[0].BuyProfitReal = buyPosition.RealizedPnl
		positions[0].BuyPriceCost = buyPosition.SettlementPrice

		positions[0].ForceLiquPrice = sellPosition.LiquidationPrice
		positions[0].SellAmount = sellPosition.Position
		positions[0].SellAvailable = sellPosition.AvailPosition
		positions[0].SellPriceAvg = sellPosition.AvgCost
		positions[0].SellProfitReal = sellPosition.RealizedPnl
		positions[0].SellPriceCost = sellPosition.SettlementPrice
	}

	//log.Println(resp)
	return positions, nil
}

/**
 * BTC: 100美元一张合约
 * LTC/ETH/ETC/BCH: 10美元一张合约
 */
func (ok *OKExSwap) GetContractValue(currencyPair CurrencyPair) (float64, error) {
	switch currencyPair {
	case BTC_USD:
		return 100, nil
	case LTC_USD, ETH_USD, ETC_USD, BCH_USD:
		return 10, nil
	}

	return -1, errors.New("error")
}

func (ok *OKExSwap) GetFee() (float64, error) {
	panic("not support")
}

func (ok *OKExSwap) GetFutureEstimatedPrice(currencyPair CurrencyPair) (float64, error) {
	panic("not support")
}

func (ok *OKExSwap) GetFutureIndex(currencyPair CurrencyPair) (float64, error) {
	panic("not support")
}

func (ok *OKExSwap) GetDeliveryTime() (int, int, int, int) {
	panic("not support")
}

func (ok *OKExSwap) GetKlineRecords(contract_type string, currency CurrencyPair, period, size, since int) ([]FutureKline, error) {
	panic("not support")
}

func (ok *OKExSwap) GetTrades(contract_type string, currencyPair CurrencyPair, since int64) ([]Trade, error) {
	panic("not support")
}

func (ok *OKExSwap) GetExchangeRate() (float64, error) {
	panic("not support")
}

func (ok *OKExSwap) doRequest(httpMethod, uri, reqBody string, response interface{}) error {
	url := Endpoint + uri
	log.Println(url)
	sign, timestamp := doParamSign(httpMethod, ok.config.ApiSecretKey, uri, reqBody)
	//log.Println(sign, timestamp)
	resp, err := NewHttpRequest(ok.config.HttpClient, httpMethod, url, reqBody, map[string]string{
		CONTENT_TYPE: APPLICATION_JSON_UTF8,
		ACCEPT:       APPLICATION_JSON,
		//COOKIE:               LOCALE + "en_US",
		OK_ACCESS_KEY:        ok.config.ApiKey,
		OK_ACCESS_PASSPHRASE: ok.config.ApiPassphrase,
		OK_ACCESS_SIGN:       sign,
		OK_ACCESS_TIMESTAMP:  fmt.Sprint(timestamp)})
	if err != nil {
		log.Println(err)
		return err
	} else {
		log.Println(string(resp))
		return json.Unmarshal(resp, &response)
	}
}

func (ok *OKExSwap) AdaptTradeStatus(status int) TradeStatus {
	switch status {
	case -1:
		return ORDER_CANCEL
	case 0:
		return ORDER_UNFINISH
	case 1:
		return ORDER_PART_FINISH
	case 2:
		return ORDER_FINISH
	default:
		return ORDER_UNFINISH
	}
}
