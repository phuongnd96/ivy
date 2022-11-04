package bigtable

import (
	"context"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"google.golang.org/api/bigtableadmin/v2"
	"google.golang.org/api/option"
)

type bigTable struct {
	project  string
	instance string
	cluster  string
	table    string
}

func NewBigtable(project, instance, cluster, table string) *bigTable {
	return &bigTable{
		project:  project,
		instance: instance,
		cluster:  cluster,
		table:    table,
	}
}

func NewTableBackup(ctx context.Context, project string, instance string, cluster string, sourceTableId string, backupId string, client *http.Client) error {
	var err error
	bigtableadminService, err := bigtableadmin.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}
	parent := fmt.Sprintf("projects/%s/instances/%s/clusters/%s", project, instance, cluster)
	table := fmt.Sprintf("projects/%s/instances/%s/tables/%s", project, instance, sourceTableId)
	log.WithFields(log.Fields{
		"project":       project,
		"instance":      instance,
		"sourceTableId": sourceTableId,
	}).Info("Creating bigtable table backup")
	_, err = bigtableadminService.Projects.Instances.Clusters.Backups.Create(parent, &bigtableadmin.Backup{
		SourceTable: table,
		// 7 Days
		ExpireTime: after(7),
	}).BackupId(backupId).Do()
	if err != nil {
		return err
	}
	return nil
}

func NewTableRestore(ctx context.Context, sourceTable bigTable, targetTable bigTable, backupId string, client *http.Client) error {
	var err error
	bigtableadminService, err := bigtableadmin.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}
	parent := getInstanceUri(targetTable.project, targetTable.instance)
	backupToRestore := getBackUpUri(sourceTable.project, sourceTable.instance, sourceTable.cluster, backupId)
	_, err = bigtableadminService.Projects.Instances.Tables.Get(getTableUri(targetTable.project, targetTable.instance, targetTable.table)).Do()
	if err != nil {
		log.Error(err)
		// Error maybe mean table is not created yet
		// TODO: more strict condition
		_, err = bigtableadminService.Projects.Instances.Tables.Restore(parent, &bigtableadmin.RestoreTableRequest{
			Backup:  backupToRestore,
			TableId: targetTable.table,
		}).Do()
		if err != nil {
			return err
		}
	}
	return nil
}

func CleanTableRestore(ctx context.Context, table bigTable, client *http.Client) error {
	var err error
	bigtableadminService, err := bigtableadmin.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"project":  table.project,
		"instance": table.instance,
		"tableId":  table.table,
	}).Info("Deleting bigtable table restore")
	tableToDelete := getTableUri(table.project, table.instance, table.table)
	_, err = bigtableadminService.Projects.Instances.Tables.Delete(tableToDelete).Do()
	if err != nil {
		return err
	}
	return nil
}
func CleanBackup(ctx context.Context, project string, instance string, cluster string, backupId string, client *http.Client) error {
	var err error
	bigtableadminService, err := bigtableadmin.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"project":  project,
		"instance": instance,
		"backupId": backupId,
	}).Info("Deleting bigtable table backup")
	backupToDelete := fmt.Sprintf("projects/%s/instances/%s/clusters/%s/backups/%s", project, instance, cluster, backupId)
	_, err = bigtableadminService.Projects.Instances.Clusters.Backups.Delete(backupToDelete).Do()
	if err != nil {
		return err
	}
	return nil
}

func after(day int) string {
	now := time.Now()
	return now.AddDate(0, 0, day).Format(time.RFC3339)
}

func getTableUri(project string, instance string, table string) string {
	return fmt.Sprintf("projects/%s/instances/%s/tables/%s", project, instance, table)
}

func getBackUpUri(project string, instance string, cluster string, backUpId string) string {
	return fmt.Sprintf("projects/%s/instances/%s/clusters/%s/backups/%s", project, instance, cluster, backUpId)
}

func getInstanceUri(project string, instance string) string {
	return fmt.Sprintf("projects/%s/instances/%s", project, instance)
}
