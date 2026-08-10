package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/payment/alipay"
	"github.com/dujiao-next/internal/payment/epay"
	"github.com/dujiao-next/internal/payment/epusdt"
	"github.com/dujiao-next/internal/payment/okpay"
	"github.com/dujiao-next/internal/payment/paypal"
	"github.com/dujiao-next/internal/payment/stripe"
	"github.com/dujiao-next/internal/payment/tokenpay"
	"github.com/dujiao-next/internal/payment/wechatpay"
	"github.com/dujiao-next/internal/payment/xunhupay"

	"github.com/shopspring/decimal"
)

// appendExchangeInfo ? payment.Amount ????????(??????????),
// ??????? ProviderPayload ???????
func appendExchangeInfo(payment *models.Payment, convertedAmount, exchangeRate, originalAmount, originalCurrency string) {
	if d, err := decimal.NewFromString(convertedAmount); err == nil {
		payment.Amount = models.Money{Decimal: d}
	}
	if payment.ProviderPayload == nil {
		payment.ProviderPayload = models.JSON{}
	}
	payment.ProviderPayload["exchange_rate"] = strings.TrimSpace(exchangeRate)
	payment.ProviderPayload["original_amount"] = originalAmount
	payment.ProviderPayload["original_currency"] = originalCurrency
}

func (s *PaymentService) applyProviderPayment(input CreatePaymentInput, order *models.Order, channel *models.PaymentChannel, payment *models.Payment) (err error) {
	providerType := strings.ToLower(strings.TrimSpace(channel.ProviderType))
	channelType := strings.ToLower(strings.TrimSpace(channel.ChannelType))
	gatewayCtx, cancel := detachOutboundRequestContext(input.Context)
	defer cancel()
	payment.GatewayOrderNo = resolveGatewayOrderNo(channel, payment)
	providerOrderNo := resolveProviderOrderNo(order.OrderNo, payment)
	log := paymentLogger(
		"order_id", order.ID,
		"order_no", order.OrderNo,
		"gateway_order_no", payment.GatewayOrderNo,
		"payment_id", payment.ID,
		"channel_id", channel.ID,
		"provider_type", providerType,
		"channel_type", channelType,
		"interaction_mode", channel.InteractionMode,
	)
	defer func() {
		if err != nil {
			log.Errorw("payment_provider_apply_failed", "error", err)
			return
		}
		log.Infow("payment_provider_apply_success")
	}()
	switch providerType {
	case constants.PaymentProviderEpay:
		if !epay.IsSupportedChannelType(channel.ChannelType) {
			return fmt.Errorf("%w: unsupported channel_type %s", ErrPaymentChannelConfigInvalid, channel.ChannelType)
		}
		cfg, err := epay.ParseConfig(channel.ConfigJSON)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		if err := epay.ValidateConfig(cfg); err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		originalAmount := payment.Amount.String()
		originalCurrency := payment.Currency
		payAmount := originalAmount
		payCurrency := originalCurrency
		if cfg.NeedsCurrencyConversion() {
			converted, targetCur, convErr := cfg.ConvertAmount(payAmount, payCurrency, 2)
			if convErr != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, convErr)
			}
			payAmount = converted
			payCurrency = targetCur
			payment.Currency = payCurrency
		}
		notifyURL := strings.TrimSpace(cfg.NotifyURL)
		returnURL := appendURLQuery(cfg.ReturnURL, buildPaymentReturnQuery(input, order, "epay_return", ""))
		subject := buildOrderSubject(order)
		createInput := epay.CreateInput{
			OrderNo:     providerOrderNo,
			Amount:      payAmount,
			Subject:     subject,
			ChannelType: channel.ChannelType,
			ClientIP:    strings.TrimSpace(input.ClientIP),
			NotifyURL:   notifyURL,
			ReturnURL:   returnURL,
		}
		if notifyURL == "" || returnURL == "" {
			return fmt.Errorf("%w: notify_url/return_url is required", ErrPaymentChannelConfigInvalid)
		}
		mode := strings.ToLower(strings.TrimSpace(channel.InteractionMode))
		if mode == constants.PaymentInteractionRedirect {
			result, err := epay.BuildRedirectURL(cfg, createInput)
			if err != nil {
				switch {
				case errors.Is(err, epay.ErrConfigInvalid), errors.Is(err, epay.ErrChannelTypeNotOK), errors.Is(err, epay.ErrSignatureGenerate):
					return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
				default:
					return ErrPaymentGatewayRequestFailed
				}
			}
			payment.PayURL = result.PayURL
			payment.QRCode = ""
			if result.Raw != nil {
				payment.ProviderPayload = models.JSON(result.Raw)
			}
			if cfg.NeedsCurrencyConversion() {
				appendExchangeInfo(payment, payAmount, cfg.ExchangeRate, originalAmount, originalCurrency)
			}
			payment.UpdatedAt = time.Now()
			if err := s.paymentRepo.Update(payment); err != nil {
				return ErrPaymentUpdateFailed
			}
			return nil
		}
		if mode != "" && mode != constants.PaymentInteractionQR {
			return ErrPaymentChannelConfigInvalid
		}
		result, err := epay.CreatePayment(gatewayCtx, cfg, createInput)
		if err != nil {
			switch {
			case errors.Is(err, epay.ErrConfigInvalid), errors.Is(err, epay.ErrChannelTypeNotOK), errors.Is(err, epay.ErrSignatureGenerate):
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			case errors.Is(err, epay.ErrRequestFailed):
				return ErrPaymentGatewayRequestFailed
			case errors.Is(err, epay.ErrResponseInvalid):
				return ErrPaymentGatewayResponseInvalid
			default:
				return ErrPaymentGatewayRequestFailed
			}
		}
		payment.PayURL = result.PayURL
		payment.QRCode = result.QRCode
		if result.TradeNo != "" {
			payment.ProviderRef = result.TradeNo
		}
		if result.Raw != nil {
			payment.ProviderPayload = models.JSON(result.Raw)
		}
		if cfg.NeedsCurrencyConversion() {
			appendExchangeInfo(payment, payAmount, cfg.ExchangeRate, originalAmount, originalCurrency)
		}
		payment.UpdatedAt = time.Now()
		if err := s.paymentRepo.Update(payment); err != nil {
			return ErrPaymentUpdateFailed
		}
		return nil
	case constants.PaymentProviderXunhupay:
		if channel.ConfigJSON == nil {
			return fmt.Errorf("%w: config is empty", ErrPaymentChannelConfigInvalid)
		}
		cfg, err := xunhupay.ParseConfig(channel.ConfigJSON)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		if err := xunhupay.ValidateConfig(cfg); err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		// ?????????? TradeType
		tradeType := resolveXunhuPayTradeType(channel.ChannelType)
		result, err := xunhupay.CreatePayment(gatewayCtx, cfg, xunhupay.CreateInput{
			OrderNo:     providerOrderNo,
			Amount:      payment.Amount.String(),
			Title:       buildOrderSubject(order),
			NotifyURL:   cfg.NotifyURL,
			ReturnURL:   appendURLQuery(cfg.ReturnURL, buildPaymentReturnQuery(input, order, "xunhupay_return", "")),
			CallbackURL: cfg.CallbackURL,
			WapName:     cfg.WapName,
			TradeType:   tradeType,
		})
		if err != nil {
			return ErrPaymentGatewayRequestFailed
		}
		payment.PayURL = result.PayURL
		payment.QRCode = result.QRCode
		if result.TradeNo != "" {
			payment.ProviderRef = result.TradeNo
		}
		if result.Raw != nil {
			payment.ProviderPayload = models.JSON(result.Raw)
		}
		payment.UpdatedAt = time.Now()
		if err := s.paymentRepo.Update(payment); err != nil {
			return ErrPaymentUpdateFailed
		}
		return nil
	case constants.PaymentProviderEpusdt:
		cfg, err := epusdt.ParseConfig(channel.ConfigJSON)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		// ????????? trade_type,?? channel_type ????
		if strings.TrimSpace(cfg.TradeType) == "" {
			cfg.TradeType = epusdt.ResolveTradeType(channel.ChannelType)
		}
		if err := epusdt.ValidateConfig(cfg); err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		notifyURL := strings.TrimSpace(cfg.NotifyURL)
		returnURL := strings.TrimSpace(cfg.ReturnURL)
		if notifyURL == "" || returnURL == "" {
			return fmt.Errorf("%w: notify_url/return_url is required", ErrPaymentChannelConfigInvalid)
		}
		returnURL = appendURLQuery(returnURL, buildPaymentReturnQuery(input, order, "epusdt_return", ""))
		subject := buildOrderSubject(order)
		result, err := epusdt.CreatePayment(gatewayCtx, cfg, epusdt.CreateInput{
			OrderNo:   providerOrderNo,
			Amount:    payment.Amount.String(),
			Name:      subject,
			NotifyURL: notifyURL,
			ReturnURL: returnURL,
		})
		if err != nil {
			switch {
			case errors.Is(err, epusdt.ErrConfigInvalid):
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			case errors.Is(err, epusdt.ErrRequestFailed):
				return ErrPaymentGatewayRequestFailed
			case errors.Is(err, epusdt.ErrResponseInvalid):
				return ErrPaymentGatewayResponseInvalid
			default:
				return ErrPaymentGatewayRequestFailed
			}
		}
		payment.PayURL = result.PaymentURL
		payment.QRCode = result.PaymentURL
		if result.TradeID != "" {
			payment.ProviderRef = result.TradeID
		}
		if result.Raw != nil {
			payment.ProviderPayload = models.JSON(result.Raw)
		}
		payment.UpdatedAt = time.Now()
		if err := s.paymentRepo.Update(payment); err != nil {
			return ErrPaymentUpdateFailed
		}
		return nil
	case constants.PaymentProviderOkpay:
		if !okpay.IsSupportedChannelType(channel.ChannelType) {
			return fmt.Errorf("%w: unsupported channel_type %s", ErrPaymentChannelConfigInvalid, channel.ChannelType)
		}
		cfg, err := okpay.ParseConfig(channel.ConfigJSON)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		if strings.TrimSpace(cfg.Coin) == "" {
			cfg.Coin = okpay.ResolveCoin(channel.ChannelType)
		}
		if err := okpay.ValidateConfig(cfg); err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		returnURL := appendURLQuery(strings.TrimSpace(cfg.ReturnURL), buildPaymentReturnQuery(input, order, "okpay_return", ""))
		createResult, err := okpay.CreatePayment(gatewayCtx, cfg, okpay.CreateInput{
			UniqueID:    providerOrderNo,
			Name:        buildOrderSubject(order),
			Amount:      payment.Amount.String(),
			ReturnURL:   returnURL,
			CallbackURL: strings.TrimSpace(cfg.CallbackURL),
			Coin:        strings.TrimSpace(cfg.Coin),
			Status:      strings.TrimSpace(cfg.Status),
		})
		if err != nil {
			switch {
			case errors.Is(err, okpay.ErrConfigInvalid):
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			case errors.Is(err, okpay.ErrRequestFailed):
				return ErrPaymentGatewayRequestFailed
			case errors.Is(err, okpay.ErrResponseInvalid):
				return ErrPaymentGatewayResponseInvalid
			default:
				return ErrPaymentGatewayRequestFailed
			}
		}
		payment.PayURL = strings.TrimSpace(createResult.PayURL)
		payment.QRCode = strings.TrimSpace(createResult.PayURL)
		payment.Status = constants.PaymentStatusPending
		payment.ProviderRef = pickFirstNonEmpty(strings.TrimSpace(createResult.OrderID), strings.TrimSpace(payment.ProviderRef), order.OrderNo)
		if createResult.Raw != nil {
			providerPayload := models.JSON(createResult.Raw)
			if convertedAmount, convertErr := okpay.ConvertAmountByRate(payment.Amount.String(), cfg.ExchangeRate); convertErr == nil {
				providerPayload["converted_amount"] = convertedAmount.StringFixed(8)
				providerPayload["exchange_rate"] = strings.TrimSpace(cfg.ExchangeRate)
			}
			payment.ProviderPayload = providerPayload
		}
		payment.UpdatedAt = time.Now()
		if err := s.paymentRepo.Update(payment); err != nil {
			return ErrPaymentUpdateFailed
		}
		return nil
	case constants.PaymentProviderTokenpay:
		cfg, err := tokenpay.ParseConfig(channel.ConfigJSON)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		if strings.TrimSpace(cfg.Currency) == "" {
			cfg.Currency = tokenpay.DefaultCurrency
		}
		if strings.TrimSpace(cfg.NotifyURL) == "" {
			return fmt.Errorf("%w: notify_url is required", ErrPaymentChannelConfigInvalid)
		}
		if err := tokenpay.ValidateConfig(cfg); err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		redirectURL := strings.TrimSpace(cfg.RedirectURL)
		if redirectURL != "" {
			redirectURL = appendURLQuery(redirectURL, buildPaymentReturnQuery(input, order, "tokenpay_return", ""))
		}
		createResult, err := tokenpay.CreatePayment(gatewayCtx, cfg, tokenpay.CreateInput{
			OutOrderID:   providerOrderNo,
			OrderUserKey: resolveTokenPayOrderUserKey(order),
			ActualAmount: payment.Amount.String(),
			Currency:     strings.TrimSpace(cfg.Currency),
			NotifyURL:    strings.TrimSpace(cfg.NotifyURL),
			RedirectURL:  redirectURL,
		})
		if err != nil {
			switch {
			case errors.Is(err, tokenpay.ErrConfigInvalid):
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			case errors.Is(err, tokenpay.ErrRequestFailed):
				return ErrPaymentGatewayRequestFailed
			case errors.Is(err, tokenpay.ErrResponseInvalid):
				return ErrPaymentGatewayResponseInvalid
			default:
				return ErrPaymentGatewayRequestFailed
			}
		}
		payment.PayURL = strings.TrimSpace(pickFirstNonEmpty(createResult.PayURL, createResult.QRCodeLink))
		payment.QRCode = strings.TrimSpace(pickFirstNonEmpty(createResult.QRCodeBase64, createResult.QRCodeLink, createResult.PayURL))
		payment.Status = constants.PaymentStatusPending
		payment.ProviderRef = pickFirstNonEmpty(strings.TrimSpace(createResult.TokenOrderID), strings.TrimSpace(payment.ProviderRef), order.OrderNo)
		if createResult.Raw != nil {
			payment.ProviderPayload = models.JSON(createResult.Raw)
		}
		payment.UpdatedAt = time.Now()
		if err := s.paymentRepo.Update(payment); err != nil {
			return ErrPaymentUpdateFailed
		}
		return nil
	case constants.PaymentProviderOfficial:
		channelType = strings.ToLower(strings.TrimSpace(channel.ChannelType))
		switch channelType {
		case constants.PaymentChannelTypePaypal:
			cfg, err := paypal.ParseConfig(channel.ConfigJSON)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			if err := paypal.ValidateConfig(cfg); err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			payAmount := payment.Amount.String()
			payCurrency := payment.Currency
			if cfg.NeedsCurrencyConversion() {
				converted, targetCur, convErr := cfg.ConvertAmount(payAmount, payCurrency)
				if convErr != nil {
					return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, convErr)
				}
				payAmount = converted
				payCurrency = targetCur
			}
			createResult, err := paypal.CreateOrder(gatewayCtx, cfg, paypal.CreateInput{
				OrderNo:     providerOrderNo,
				Amount:      payAmount,
				Currency:    payCurrency,
				Description: buildOrderSubject(order),
				ReturnURL:   appendURLQuery(cfg.ReturnURL, buildPaymentReturnQuery(input, order, "pp_return", "")),
				CancelURL:   appendURLQuery(cfg.CancelURL, buildPaymentReturnQuery(input, order, "pp_cancel", "")),
			})
			if err != nil {
				switch {
				case errors.Is(err, paypal.ErrConfigInvalid):
					return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
				case errors.Is(err, paypal.ErrAuthFailed), errors.Is(err, paypal.ErrRequestFailed):
					return ErrPaymentGatewayRequestFailed
				case errors.Is(err, paypal.ErrResponseInvalid):
					return ErrPaymentGatewayResponseInvalid
				default:
					return ErrPaymentGatewayRequestFailed
				}
			}
			payment.PayURL = strings.TrimSpace(createResult.ApprovalURL)
			payment.QRCode = ""
			payment.Status = constants.PaymentStatusPending
			payment.ProviderRef = strings.TrimSpace(createResult.OrderID)
			if createResult.Raw != nil {
				payment.ProviderPayload = models.JSON(createResult.Raw)
			}
			if cfg.NeedsCurrencyConversion() {
				appendExchangeInfo(payment, payAmount, cfg.ExchangeRate, payment.Amount.String(), payment.Currency)
				payment.Currency = payCurrency
			}
			payment.UpdatedAt = time.Now()
			if err := s.paymentRepo.Update(payment); err != nil {
				return ErrPaymentUpdateFailed
			}
			return nil
		case constants.PaymentChannelTypeAlipay:
			cfg, err := alipay.ParseConfig(channel.ConfigJSON)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			if err := alipay.ValidateConfig(cfg, channel.InteractionMode); err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			originalAmount := payment.Amount.String()
			originalCurrency := payment.Currency
			payAmount := originalAmount
			payCurrency := originalCurrency
			if cfg.NeedsCurrencyConversion() {
				converted, targetCur, convErr := cfg.ConvertAmount(payAmount, payCurrency, 2)
				if convErr != nil {
					return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, convErr)
				}
				payAmount = converted
				payCurrency = targetCur
			} else {
				payCurrency = "CNY"
			}
			payment.Currency = payCurrency
			createResult, err := alipay.CreatePayment(gatewayCtx, cfg, alipay.CreateInput{
				OrderNo:   providerOrderNo,
				Amount:    payAmount,
				Subject:   "???????",
				NotifyURL: cfg.NotifyURL,
				ReturnURL: appendURLQuery(cfg.ReturnURL, buildPaymentReturnQuery(input, order, "alipay_return", "")),
			}, channel.InteractionMode)
			if err != nil {
				switch {
				case errors.Is(err, alipay.ErrConfigInvalid), errors.Is(err, alipay.ErrSignGenerate):
					return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
				case errors.Is(err, alipay.ErrRequestFailed):
					return ErrPaymentGatewayRequestFailed
				case errors.Is(err, alipay.ErrResponseInvalid):
					return ErrPaymentGatewayResponseInvalid
				default:
					return ErrPaymentGatewayRequestFailed
				}
			}
			payment.PayURL = strings.TrimSpace(createResult.PayURL)
			payment.QRCode = strings.TrimSpace(createResult.QRCode)
			payment.Status = constants.PaymentStatusPending
			payment.ProviderRef = pickFirstNonEmpty(strings.TrimSpace(createResult.TradeNo), strings.TrimSpace(createResult.OutTradeNo), order.OrderNo)
			if createResult.Raw != nil {
				payment.ProviderPayload = models.JSON(createResult.Raw)
			}
			if cfg.NeedsCurrencyConversion() {
				appendExchangeInfo(payment, payAmount, cfg.ExchangeRate, originalAmount, originalCurrency)
			}
			payment.UpdatedAt = time.Now()
			if err := s.paymentRepo.Update(payment); err != nil {
				return ErrPaymentUpdateFailed
			}
			return nil
		case constants.PaymentChannelTypeWechat:
			cfg, err := wechatpay.ParseConfig(channel.ConfigJSON)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			if err := wechatpay.ValidateConfig(cfg, channel.InteractionMode); err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			originalAmount := payment.Amount.String()
			originalCurrency := payment.Currency
			payAmount := originalAmount
			payCurrency := originalCurrency
			if cfg.NeedsCurrencyConversion() {
				converted, targetCur, convErr := cfg.ConvertAmount(payAmount, payCurrency, 2)
				if convErr != nil {
					return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, convErr)
				}
				payAmount = converted
				payCurrency = targetCur
			} else {
				payCurrency = "CNY"
			}
			payment.Currency = payCurrency
			cfgForCreate := *cfg
			cfgForCreate.H5RedirectURL = appendURLQuery(cfg.H5RedirectURL, buildPaymentReturnQuery(input, order, "wechat_return", ""))
			createResult, err := wechatpay.CreatePayment(gatewayCtx, &cfgForCreate, wechatpay.CreateInput{
				OrderNo:     providerOrderNo,
				Amount:      payAmount,
				Currency:    payCurrency,
				Description: buildOrderSubject(order),
				ClientIP:    strings.TrimSpace(input.ClientIP),
				NotifyURL:   cfg.NotifyURL,
			}, channel.InteractionMode)
			if err != nil {
				switch {
				case errors.Is(err, wechatpay.ErrConfigInvalid):
					return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
				case errors.Is(err, wechatpay.ErrRequestFailed):
					return ErrPaymentGatewayRequestFailed
				case errors.Is(err, wechatpay.ErrResponseInvalid):
					return ErrPaymentGatewayResponseInvalid
				default:
					return ErrPaymentGatewayRequestFailed
				}
			}
			payment.PayURL = strings.TrimSpace(createResult.PayURL)
			payment.QRCode = strings.TrimSpace(createResult.QRCode)
			payment.Status = constants.PaymentStatusPending
			payment.ProviderRef = pickFirstNonEmpty(strings.TrimSpace(payment.ProviderRef), order.OrderNo)
			if createResult.Raw != nil {
				payment.ProviderPayload = models.JSON(createResult.Raw)
			}
			if cfg.NeedsCurrencyConversion() {
				appendExchangeInfo(payment, payAmount, cfg.ExchangeRate, originalAmount, originalCurrency)
			}
			payment.UpdatedAt = time.Now()
			if err := s.paymentRepo.Update(payment); err != nil {
				return ErrPaymentUpdateFailed
			}
			return nil
		case constants.PaymentChannelTypeStripe:
			cfg, err := stripe.ParseConfig(channel.ConfigJSON)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			if err := stripe.ValidateConfig(cfg); err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			originalAmount := payment.Amount.String()
			originalCurrency := payment.Currency
			payAmount := originalAmount
			payCurrency := originalCurrency
			if cfg.NeedsCurrencyConversion() {
				converted, targetCur, convErr := cfg.ConvertAmount(payAmount, payCurrency, 2)
				if convErr != nil {
					return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, convErr)
				}
				payAmount = converted
				payCurrency = targetCur
				payment.Currency = payCurrency
			}
			createResult, err := stripe.CreatePayment(gatewayCtx, cfg, stripe.CreateInput{
				OrderNo:     providerOrderNo,
				Amount:      payAmount,
				Currency:    payCurrency,
				Description: buildOrderSubject(order),
				SuccessURL:  appendURLQuery(cfg.SuccessURL, buildPaymentReturnQuery(input, order, "stripe_return", "{CHECKOUT_SESSION_ID}")),
				CancelURL:   appendURLQuery(cfg.CancelURL, buildPaymentReturnQuery(input, order, "stripe_cancel", "")),
			})
			if err != nil {
				return mapStripeGatewayError(err)
			}
			payment.PayURL = strings.TrimSpace(createResult.URL)
			payment.QRCode = ""
			payment.Status = constants.PaymentStatusPending
			payment.ProviderRef = pickFirstNonEmpty(strings.TrimSpace(createResult.SessionID), strings.TrimSpace(createResult.PaymentIntentID), order.OrderNo)
			if createResult.Raw != nil {
				payment.ProviderPayload = models.JSON(createResult.Raw)
			}
			if cfg.NeedsCurrencyConversion() {
				appendExchangeInfo(payment, payAmount, cfg.ExchangeRate, originalAmount, originalCurrency)
			}
			payment.UpdatedAt = time.Now()
			if err := s.paymentRepo.Update(payment); err != nil {
				return ErrPaymentUpdateFailed
			}
			return nil
		default:
			return ErrPaymentProviderNotSupported
		}
	default:
		return ErrPaymentProviderNotSupported
	}
}

// ValidateChannel ????????
func (s *PaymentService) ValidateChannel(channel *models.PaymentChannel) error {
	if channel == nil {
		return ErrPaymentChannelConfigInvalid
	}
	feeRate := channel.FeeRate.Decimal.Round(2)
	if feeRate.LessThan(decimal.Zero) || feeRate.GreaterThan(decimal.NewFromInt(100)) {
		return ErrPaymentChannelConfigInvalid
	}
	fixedFee := channel.FixedFee.Decimal.Round(2)
	// decimal(6,2) max value is 9999.99
	if fixedFee.LessThan(decimal.Zero) || fixedFee.GreaterThanOrEqual(decimal.NewFromInt(10000)) {
		return ErrPaymentChannelConfigInvalid
	}
	minAmount := channel.MinAmount.Decimal.Round(2)
	maxAmount := channel.MaxAmount.Decimal.Round(2)
	amountOverflow20_2 := decimal.NewFromInt(1000000000000000000)
	// min/max amount are stored as decimal(20,2), max allowed is 999999999999999999.99.
	if minAmount.LessThan(decimal.Zero) || minAmount.GreaterThanOrEqual(amountOverflow20_2) || maxAmount.LessThan(decimal.Zero) || maxAmount.GreaterThanOrEqual(amountOverflow20_2) {
		return ErrPaymentChannelConfigInvalid
	}
	if maxAmount.GreaterThan(decimal.Zero) && minAmount.GreaterThan(maxAmount) {
		return ErrPaymentChannelConfigInvalid
	}
	providerType := strings.ToLower(strings.TrimSpace(channel.ProviderType))
	switch providerType {
	case constants.PaymentProviderEpay:
		if !epay.IsSupportedChannelType(channel.ChannelType) {
			return fmt.Errorf("%w: unsupported channel_type %s", ErrPaymentChannelConfigInvalid, channel.ChannelType)
		}
		mode := strings.ToLower(strings.TrimSpace(channel.InteractionMode))
		if mode != constants.PaymentInteractionQR && mode != constants.PaymentInteractionRedirect {
			return ErrPaymentChannelConfigInvalid
		}
		cfg, err := epay.ParseConfig(channel.ConfigJSON)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		if err := epay.ValidateConfig(cfg); err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		return nil
	case constants.PaymentProviderEpusdt:
		if !epusdt.IsSupportedChannelType(channel.ChannelType) {
			return fmt.Errorf("%w: unsupported channel_type %s", ErrPaymentChannelConfigInvalid, channel.ChannelType)
		}
		if strings.ToLower(strings.TrimSpace(channel.InteractionMode)) != constants.PaymentInteractionRedirect &&
			strings.ToLower(strings.TrimSpace(channel.InteractionMode)) != constants.PaymentInteractionQR {
			return ErrPaymentChannelConfigInvalid
		}
		cfg, err := epusdt.ParseConfig(channel.ConfigJSON)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		if err := epusdt.ValidateConfig(cfg); err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		return nil
	case constants.PaymentProviderOkpay:
		if !okpay.IsSupportedChannelType(channel.ChannelType) {
			return fmt.Errorf("%w: unsupported channel_type %s", ErrPaymentChannelConfigInvalid, channel.ChannelType)
		}
		mode := strings.ToLower(strings.TrimSpace(channel.InteractionMode))
		if mode != constants.PaymentInteractionQR && mode != constants.PaymentInteractionRedirect {
			return ErrPaymentChannelConfigInvalid
		}
		cfg, err := okpay.ParseConfig(channel.ConfigJSON)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		if strings.TrimSpace(cfg.Coin) == "" {
			cfg.Coin = okpay.ResolveCoin(channel.ChannelType)
		}
		if err := okpay.ValidateConfig(cfg); err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		return nil
	case constants.PaymentProviderTokenpay:
		if strings.ToLower(strings.TrimSpace(channel.InteractionMode)) != constants.PaymentInteractionRedirect &&
			strings.ToLower(strings.TrimSpace(channel.InteractionMode)) != constants.PaymentInteractionQR {
			return ErrPaymentChannelConfigInvalid
		}
		cfg, err := tokenpay.ParseConfig(channel.ConfigJSON)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		if strings.TrimSpace(cfg.Currency) == "" {
			cfg.Currency = tokenpay.DefaultCurrency
		}
		if strings.TrimSpace(cfg.NotifyURL) == "" {
			return fmt.Errorf("%w: notify_url is required", ErrPaymentChannelConfigInvalid)
		}
		if err := tokenpay.ValidateConfig(cfg); err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		return nil
	case constants.PaymentProviderOfficial:
		channelType := strings.ToLower(strings.TrimSpace(channel.ChannelType))
		switch channelType {
		case constants.PaymentChannelTypePaypal:
			if strings.ToLower(strings.TrimSpace(channel.InteractionMode)) != constants.PaymentInteractionRedirect {
				return ErrPaymentChannelConfigInvalid
			}
			cfg, err := paypal.ParseConfig(channel.ConfigJSON)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			if err := paypal.ValidateConfig(cfg); err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			return nil
		case constants.PaymentChannelTypeAlipay:
			cfg, err := alipay.ParseConfig(channel.ConfigJSON)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			if err := alipay.ValidateConfig(cfg, channel.InteractionMode); err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			return nil
		case constants.PaymentChannelTypeWechat:
			cfg, err := wechatpay.ParseConfig(channel.ConfigJSON)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			if err := wechatpay.ValidateConfig(cfg, channel.InteractionMode); err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			return nil
		case constants.PaymentChannelTypeStripe:
			if strings.ToLower(strings.TrimSpace(channel.InteractionMode)) != constants.PaymentInteractionRedirect {
				return ErrPaymentChannelConfigInvalid
			}
			cfg, err := stripe.ParseConfig(channel.ConfigJSON)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			if err := stripe.ValidateConfig(cfg); err != nil {
				return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			}
			return nil
		default:
			return ErrPaymentProviderNotSupported
		}
	case constants.PaymentProviderXunhupay:
		mode := strings.ToLower(strings.TrimSpace(channel.InteractionMode))
		if mode != constants.PaymentInteractionQR && mode != constants.PaymentInteractionRedirect {
			return ErrPaymentChannelConfigInvalid
		}
		cfg, err := xunhupay.ParseConfig(channel.ConfigJSON)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		if err := xunhupay.ValidateConfig(cfg); err != nil {
			return fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		}
		return nil
	default:
		return ErrPaymentProviderNotSupported
	}
}

func mapPaypalStatus(status string) (string, bool) {
	status = strings.ToUpper(strings.TrimSpace(status))
	switch status {
	case "COMPLETED":
		return constants.PaymentStatusSuccess, true
	case "PENDING", "APPROVED", "CREATED", "SAVED":
		return constants.PaymentStatusPending, true
	case "DECLINED", "DENIED", "FAILED", "VOIDED":
		return constants.PaymentStatusFailed, true
	default:
		return "", false
	}
}

func resolveTokenPayOrderUserKey(order *models.Order) string {
	if order == nil {
		return ""
	}
	if order.UserID > 0 {
		return strconv.FormatUint(uint64(order.UserID), 10)
	}
	if guestEmail := strings.TrimSpace(order.GuestEmail); guestEmail != "" {
		return guestEmail
	}
	return strings.TrimSpace(order.OrderNo)
}

// resolveXunhuPayTradeType ??????????????
func resolveXunhuPayTradeType(channelType string) string {
	ct := strings.ToLower(strings.TrimSpace(channelType))
	switch ct {
	case constants.PaymentChannelTypeWechat, constants.PaymentChannelTypeWxpay:
		return "wechat"
	case constants.PaymentChannelTypeAlipay:
		return "alipay"
	case constants.PaymentChannelTypeQqpay:
		return "qqpay"
	default:
		return "wechat"
	}
}
