package routes

import (
	"photography-server/internal/domain"
	"photography-server/internal/presentation/controller"
)

// catalog 套餐（status 即上下架）。
func catalog(ctl *controller.Controller) []Route {
	return []Route{
		{Path: "/package/list", Perm: domain.PermPackageView, Handler: ctl.PackageList},
		{Path: "/package/detail/:id", Perm: domain.PermPackageView, Handler: ctl.PackageDetail},
		{Path: "/package/create", Perm: domain.PermPackageCreate, Handler: ctl.PackageCreate},
		{Path: "/package/update/:id", Perm: domain.PermPackageUpdate, Handler: ctl.PackageUpdate},
		{Path: "/package/status/:id", Perm: domain.PermPackagePublish, Handler: ctl.PackageStatus},
		{Path: "/package/delete/:id", Perm: domain.PermPackageDelete, Handler: ctl.PackageDelete},
	}
}
