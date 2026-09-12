package routes

import (
	"photography-server/internal/domain"
	"photography-server/internal/presentation/controller"
)

// paymentRefundDelivery 收款 / 退款 / 交付 / 作品集。
//
// 收款（登记与核验到账分离：登记不改变资金确认状态，核验才是）；
// 退款（发起与审批分离：发起者不得自审）；
// 交付（读接口归 view，上传/选片/确认等推进动作归 update）；
// 作品集 status 接口控制可见性/精选，属"发布审核"动作 → asset:audit，
// 故摄影师可上传/编辑自己的作品，但无权决定是否公开——需店长审核。
func paymentRefundDelivery(ctl *controller.Controller) []Route {
	return []Route{
		{Path: "/payment/create/:order_id", Perm: domain.PermPaymentCreate, Handler: ctl.PaymentCreate},
		{Path: "/payment/list/:order_id", Perm: domain.PermPaymentView, Handler: ctl.PaymentList},
		{Path: "/payment/confirm/:id", Perm: domain.PermPaymentConfirm, Handler: ctl.PaymentConfirm},
		{Path: "/payment/delete/:id", Perm: domain.PermPaymentDelete, Handler: ctl.PaymentDelete},

		{Path: "/refund/apply/:order_id", Perm: domain.PermRefundCreate, Handler: ctl.RefundApply},
		{Path: "/refund/list/:order_id", Perm: domain.PermRefundView, Handler: ctl.RefundList},
		{Path: "/refund/audit/:id", Perm: domain.PermRefundAudit, Handler: ctl.RefundAudit},

		{Path: "/delivery/list", Perm: domain.PermDeliveryView, Handler: ctl.DeliveryList},
		{Path: "/delivery/create/:order_id", Perm: domain.PermDeliveryCreate, Handler: ctl.DeliveryCreate},
		{Path: "/delivery/remind/:id", Perm: domain.PermDeliveryUpdate, Handler: ctl.DeliveryRemind},
		{Path: "/delivery/detail/:id", Perm: domain.PermDeliveryView, Handler: ctl.DeliveryDetail},
		{Path: "/delivery/items/:id", Perm: domain.PermDeliveryView, Handler: ctl.DeliveryItems},
		{Path: "/delivery/upload-samples/:id", Perm: domain.PermDeliveryUpdate, Handler: ctl.DeliveryUploadSamples},
		{Path: "/delivery/select/:id", Perm: domain.PermDeliveryUpdate, Handler: ctl.DeliverySelect},
		{Path: "/delivery/upload-retouched/:id", Perm: domain.PermDeliveryUpdate, Handler: ctl.DeliveryUploadRetouched},
		{Path: "/delivery/confirm/:id", Perm: domain.PermDeliveryUpdate, Handler: ctl.DeliveryConfirm},

		{Path: "/asset/list", Perm: domain.PermAssetView, Handler: ctl.AssetList},
		{Path: "/asset/detail/:id", Perm: domain.PermAssetView, Handler: ctl.AssetDetail},
		{Path: "/asset/create", Perm: domain.PermAssetUpload, Handler: ctl.AssetCreate},
		{Path: "/asset/update/:id", Perm: domain.PermAssetUpdate, Handler: ctl.AssetUpdate},
		{Path: "/asset/status/:id", Perm: domain.PermAssetAudit, Handler: ctl.AssetStatus},
		{Path: "/asset/delete/:id", Perm: domain.PermAssetDelete, Handler: ctl.AssetDelete},
	}
}
