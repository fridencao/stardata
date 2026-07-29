package river

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/fridencao/stardata/admin"
	"github.com/riverqueue/river"
	"golang.org/x/sync/errgroup"
)

const _unusedAssetsPageSize = 100

type DeleteUnusedAssetsArgs struct{}

func (DeleteUnusedAssetsArgs) Kind() string { return "delete_unused_assets" }

type DeleteUnusedAssetsWorker struct {
	river.WorkerDefaults[DeleteUnusedAssetsArgs]
	admin *admin.Service
}

func (w *DeleteUnusedAssetsWorker) Work(ctx context.Context, job *river.Job[DeleteUnusedAssetsArgs]) error {
	if w.admin.Assets == nil {
		return nil
	}
	for {
		// 1. Fetch unused assets
		assets, err := w.admin.DB.FindUnusedAssets(ctx, _unusedAssetsPageSize)
		if err != nil {
			return err
		}
		if len(assets) == 0 {
			return nil
		}

		// 2. Delete objects from cloud storage
		// Limit the number of concurrent deletes to 8
		// TODO: Use batch API once google-cloud-go supports it
		group, cctx := errgroup.WithContext(ctx)
		group.SetLimit(8)
		var ids []string
		for _, asset := range assets {
			asset := asset
			ids = append(ids, asset.ID)
			group.Go(func() error {
				parsed, err := url.Parse(asset.Path)
				if err != nil {
					return fmt.Errorf("failed to parse asset path %q: %w", asset.Path, err)
				}
				err = w.admin.Assets.Delete(cctx, strings.TrimPrefix(parsed.Path, "/"))
				if err != nil {
					return fmt.Errorf("failed to delete asset %q: %w", asset.Path, err)
				}
				return nil
			})
		}
		err = group.Wait()
		if err != nil {
			return err
		}

		// 3. Delete the assets in the DB
		err = w.admin.DB.DeleteAssets(ctx, ids)
		if err != nil {
			return err
		}

		if len(assets) < _unusedAssetsPageSize {
			// no more assets to delete
			return nil
		}
		// fetch again could be more unused assets
	}
}
