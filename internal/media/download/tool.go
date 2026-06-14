package download

import (
	"context"

	"github.com/OpenListTeam/OpenList/v4/internal/offline_download/tool"
)

func NewOpenListToolDownloader() OpenListDownloader {
	return OpenListDownloader{
		AddURL: func(ctx context.Context, args AddURLArgs) (*TaskRef, error) {
			info, err := tool.AddURL(ctx, &tool.AddURLArgs{
				URL:        args.URL,
				DstDirPath: args.DstDirPath,
				Tool:       args.Tool,
			})
			if err != nil {
				return nil, err
			}
			return TaskRefFromInfo(info)
		},
	}
}
