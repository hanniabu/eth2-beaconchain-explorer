package main

import (
	"eth2-exporter/db"
	"eth2-exporter/types"
	"eth2-exporter/utils"
	"flag"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/sirupsen/logrus"
	"time"
)

func main() {
	configPath := flag.String("config", "", "Path to the config file")
	flag.Parse()

	logrus.Printf("config file path: %v", *configPath)
	cfg := &types.Config{}
	err := utils.ReadConfig(cfg, *configPath)

	if err != nil {
		logrus.Fatalf("error reading config file: %v", err)
	}
	utils.Config = cfg

	db.MustInitDB(cfg.Database.Username, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	defer db.DB.Close()

	for true {
		latestEpoch, err := db.GetLatestEpoch()
		if err != nil {
			logrus.Errorf("error retreiving latest epoch from the db: %v", err)
		}
		currentDay := latestEpoch / ((24 * 60 * 60) / utils.Config.Chain.SlotsPerEpoch / utils.Config.Chain.SecondsPerSlot)
		previousDay := currentDay - 1

		for day := uint64(0); day <= previousDay; day++ {
			var status bool
			err := db.DB.Get(&status, "select status from validator_stats_status where day = $1", day)
			if err != nil && err.Error() == "sql: no rows in result set" {
				err := db.WriteStatisticsForDay(day)
				if err != nil {
					logrus.Errorf("error exporting stats for day %v: %v", day, err)
				}
			} else if err != nil {
				logrus.Errorf("error retrieving stats status for day %v: %v", day, err)
			} else {
				logrus.Infof("statistics for day %v are already in the db", day)
			}
		}

		time.Sleep(time.Hour)
	}
}
