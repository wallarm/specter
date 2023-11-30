package coreutil

import (
	"go.uber.org/zap"
	"reflect"

	"github.com/wallarm/specter/core"
)

func ResetReusedAmmo(ammo core.Ammo) {
	logger := zap.L().Named("ResetReusedAmmo")

	if resettable, ok := ammo.(core.ResettableAmmo); ok {
		logger.Info("Ammo is ResettableAmmo", zap.String("type", reflect.TypeOf(ammo).String()))
		resettable.Reset()
		return
	}

	val := reflect.ValueOf(ammo)
	logger.Info("Processing non-ResettableAmmo", zap.String("type", reflect.TypeOf(ammo).String()))

	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.Elem().CanSet() {
			elem := val.Elem()
			elem.Set(reflect.Zero(elem.Type()))
			logger.Info("Ammo reset to zero", zap.String("type", elem.Type().String()))
		} else {
			logger.Info("Ammo cannot be set to zero", zap.String("type", val.Elem().Type().String()))
		}
	} else {
		logger.Info("Invalid type for Elem", zap.String("type", val.Type().String()))
	}
}
