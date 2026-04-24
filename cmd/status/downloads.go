package statuscmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/seekthought/skill/pkg/db"

	cmdutils "github.com/seekthought/skill/cmd/utils"

	"github.com/spf13/cobra"
)

func newDownloadsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "downloads [skill-name]",
		Short: "Show recorded skill install history",
		Long:  `Display recorded local install/download history for skills. Provide a skill name to filter the history.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillName := ""
			if len(args) > 0 {
				skillName = args[0]
			}
			return runStatusDownloads(skillName)
		},
	}
}

func runStatusDownloads(skillName string) error {
	manager, err := cmdutils.InitDB()
	if err != nil {
		return err
	}
	defer cmdutils.CloseDB(manager)

	ctx := context.Background()
	var records []db.SkillDownload
	if skillName != "" {
		records, err = manager.GetQueries().ListSkillDownloadsBySkill(ctx, db.ListSkillDownloadsBySkillParams{
			SkillName:   skillName,
			InstalledAs: skillName,
		})
	} else {
		records, err = manager.GetQueries().ListSkillDownloads(ctx)
	}
	if err != nil {
		return fmt.Errorf("failed to list download history: %w", err)
	}

	if skillName == "" {
		fmt.Println("Skill Download History:")
	} else {
		fmt.Printf("Skill Download History: %s\n", skillName)
	}
	fmt.Println()

	if len(records) == 0 {
		fmt.Println("No download history found.")
		return nil
	}

	for i, record := range records {
		fmt.Printf("%d. %s\n", i+1, record.SkillName)
		if record.InstalledAs != "" && record.InstalledAs != record.SkillName {
			fmt.Printf("   Installed As: %s\n", record.InstalledAs)
		}
		if record.SkillVersion.Valid && record.SkillVersion.String != "" {
			fmt.Printf("   Version: %s\n", record.SkillVersion.String)
		}
		fmt.Printf("   Source: %s\n", formatDownloadSource(record))
		fmt.Printf("   Downloader: %s\n", formatDownloader(record))
		if record.DownloadedAt != "" {
			if downloadedAt, err := time.Parse(time.RFC3339, record.DownloadedAt); err == nil {
				fmt.Printf("   Downloaded: %s\n", downloadedAt.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("   Downloaded: %s\n", record.DownloadedAt)
			}
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d record(s)\n", len(records))
	return nil
}

func formatDownloadSource(record db.SkillDownload) string {
	parts := []string{record.SourceType}
	if record.SourceName.Valid && record.SourceName.String != "" {
		parts = append(parts, record.SourceName.String)
	}
	if record.SourceUrl.Valid && record.SourceUrl.String != "" {
		parts = append(parts, record.SourceUrl.String)
	}
	return strings.Join(parts, " | ")
}

func formatDownloader(record db.SkillDownload) string {
	if record.DownloaderHost.Valid && record.DownloaderHost.String != "" {
		return fmt.Sprintf("%s@%s", record.Downloader, record.DownloaderHost.String)
	}
	return record.Downloader
}
