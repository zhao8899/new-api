package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestGetTopUpReconciliationSummaryGroupsAndFilters(t *testing.T) {
	truncateTables(t)

	rows := []*TopUp{
		{
			UserId:          1,
			Amount:          10,
			Money:           73,
			TradeNo:         "reconcile-alipay-success-1",
			PaymentMethod:   "alipay",
			PaymentProvider: PaymentProviderEpay,
			CreateTime:      100,
			Status:          common.TopUpStatusSuccess,
		},
		{
			UserId:          1,
			Amount:          20,
			Money:           146,
			TradeNo:         "reconcile-alipay-success-2",
			PaymentMethod:   "alipay",
			PaymentProvider: PaymentProviderEpay,
			CreateTime:      110,
			Status:          common.TopUpStatusSuccess,
		},
		{
			UserId:          1,
			Amount:          30,
			Money:           219,
			TradeNo:         "reconcile-wxpay-pending",
			PaymentMethod:   "wxpay",
			PaymentProvider: PaymentProviderEpay,
			CreateTime:      120,
			Status:          common.TopUpStatusPending,
		},
		{
			UserId:          1,
			Amount:          40,
			Money:           292,
			TradeNo:         "reconcile-old",
			PaymentMethod:   "alipay",
			PaymentProvider: PaymentProviderEpay,
			CreateTime:      10,
			Status:          common.TopUpStatusSuccess,
		},
	}

	for _, row := range rows {
		require.NoError(t, row.Insert())
	}

	got, err := GetTopUpReconciliationSummary(TopUpReconciliationQuery{
		StartTime:       90,
		EndTime:         130,
		PaymentProvider: PaymentProviderEpay,
	})
	require.NoError(t, err)
	require.Len(t, got, 2)

	require.Equal(t, "alipay", got[0].PaymentMethod)
	require.Equal(t, common.TopUpStatusSuccess, got[0].Status)
	require.Equal(t, int64(2), got[0].OrderCount)
	require.Equal(t, int64(30), got[0].TotalAmount)
	require.InDelta(t, 219, got[0].TotalMoney, 0.001)

	require.Equal(t, "wxpay", got[1].PaymentMethod)
	require.Equal(t, common.TopUpStatusPending, got[1].Status)
	require.Equal(t, int64(1), got[1].OrderCount)
	require.Equal(t, int64(30), got[1].TotalAmount)
	require.InDelta(t, 219, got[1].TotalMoney, 0.001)

	successOnly, err := GetTopUpReconciliationSummary(TopUpReconciliationQuery{
		StartTime:     90,
		EndTime:       130,
		PaymentMethod: "alipay",
		Status:        common.TopUpStatusSuccess,
	})
	require.NoError(t, err)
	require.Len(t, successOnly, 1)
	require.Equal(t, int64(2), successOnly[0].OrderCount)
}
