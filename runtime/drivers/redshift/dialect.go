package redshift

import (
	"github.com/fridencao/stardata/runtime/drivers"
)

type dialect struct {
	drivers.BaseDialect
}

var DialectRedshift drivers.Dialect = func() drivers.Dialect {
	d := &dialect{}
	d.BaseDialect = drivers.NewBaseDialect(drivers.DialectNameRedshift, drivers.DoubleQuotesEscapeIdentifier, drivers.DoubleQuotesEscapeIdentifier)
	return d
}()
