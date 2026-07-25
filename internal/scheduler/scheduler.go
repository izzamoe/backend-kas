package scheduler

import (
	"os"

	"github.com/pocketbase/pocketbase/core"

	"kas/internal/service"
)

func Register(app core.App, cronService service.DigiflazzCronService) {
	priceSyncSchedule := os.Getenv("DIGIFLAZZ_PRICE_SYNC_INTERVAL")
	if priceSyncSchedule == "" {
		priceSyncSchedule = "*/30 * * * *"
	}
	orderPollSchedule := os.Getenv("DIGIFLAZZ_ORDER_POLL_INTERVAL")
	if orderPollSchedule == "" {
		orderPollSchedule = "*/5 * * * *"
	}

	app.Cron().MustAdd("digiflazz-price-sync", priceSyncSchedule, cronService.RunPriceSync)
	app.Cron().MustAdd("digiflazz-order-poll", orderPollSchedule, cronService.RunOrderPoll)
}
