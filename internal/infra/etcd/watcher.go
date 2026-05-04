package etcd

import (
	"context"
	"log/slog"

	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"

	"github.com/jva44ka/marketplace-simulator-cart/internal/infra/config"
)

// Watch подписывается на изменения ключа в etcd. При изменении обновляет ConfigStore
// и вызывает onChange callback. Блокирует до отмены ctx.
func Watch(
	ctx context.Context,
	client *clientv3.Client,
	key string,
	store *config.ConfigStore,
	onChange func(*config.Config),
) {
	watchChan := client.Watch(ctx, key)
	slog.Info("etcd: watching config key", "key", key)

	for {
		select {
		case <-ctx.Done():
			return
		case resp, ok := <-watchChan:
			if !ok {
				slog.Warn("etcd: watch channel closed")
				return
			}
			if resp.Err() != nil {
				slog.Error("etcd: watch error", "err", resp.Err())
				continue
			}
			for _, ev := range resp.Events {
				if ev.Type != clientv3.EventTypePut {
					continue
				}

				var newCfg config.Config
				if err := yaml.Unmarshal(ev.Kv.Value, &newCfg); err != nil {
					slog.Error("etcd: failed to parse config update", "key", key, "err", err)
					continue
				}

				warnIfRestartRequired(store.Load(), &newCfg)
				store.Store(&newCfg)
				slog.Info("etcd: config updated from etcd", "key", key)

				if onChange != nil {
					onChange(&newCfg)
				}
			}
		}
	}
}

// warnIfRestartRequired логирует предупреждение если изменились поля,
// которые применяются только при перезапуске сервиса.
func warnIfRestartRequired(old, new_ *config.Config) {
	if old.Database != new_.Database {
		slog.Warn("etcd: database config changed — restart required to take effect")
	}
	if old.Server != new_.Server {
		slog.Warn("etcd: server address changed — restart required to take effect")
	}
	if old.Tracing != new_.Tracing {
		slog.Warn("etcd: tracing config changed — restart required to take effect")
	}
	if old.Products.Host != new_.Products.Host ||
		old.Products.Port != new_.Products.Port ||
		old.Products.Schema != new_.Products.Schema {
		slog.Warn("etcd: products connection config changed — restart required to take effect")
	}
}
