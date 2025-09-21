package httpx

import (
	"SepTaf/internal/config"
	mdb "SepTaf/internal/mongo"
)

var (
	depMC  *mdb.Client
	depCfg config.Config
)

func SetDeps(mc *mdb.Client, cfg config.Config) {
	depMC = mc
	depCfg = cfg
}
