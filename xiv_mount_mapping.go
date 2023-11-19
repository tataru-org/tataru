package main

import "fmt"

type BossID string
type BossName string

type MountID string
type MountName string
type Mount struct {
	ID   MountID
	Name MountName
}

func getXivMountMetadata() ([]*Mount, error) {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("database connection acquire error: [%w]", err)
	}
	defer dbcon.Release()
	query := `
		select
			mount_id,
			mount_name
		from bot.mount_metadata
	`
	rows, err := dbcon.Query(
		ctx,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("get mount metadata error: [%w]", err)
	}
	mounts := []*Mount{}
	for rows.Next() {
		var mountID string
		var mountName string
		err = rows.Scan(&mountID, &mountName)
		if err != nil {
			return nil, fmt.Errorf("row scan error: [%w]", err)
		}
		mounts = append(
			mounts,
			&Mount{
				ID:   MountID(mountID),
				Name: MountName(mountName),
			},
		)
	}
	return mounts, nil
}

func getXivBossMountMapping() (map[BossName]MountName, error) {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("database connection acquire error: [%w]", err)
	}
	defer dbcon.Release()
	query := `
		select
			b.boss_name,
			d.mount_name
		from bot.boss_metadata b
		inner join bot.boss_mount_map m
		on b.boss_id = m.boss_id
		inner join bot.mount_metadata d
		on d.mount_id = m.mount_id
	`
	rows, err := dbcon.Query(
		ctx,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("get boss and mount metadata error: [%w]", err)
	}
	bossMountMap := make(map[BossName]MountName)
	for rows.Next() {
		var bossName string
		var mountName string
		err = rows.Scan(&bossName, &mountName)
		if err != nil {
			return nil, fmt.Errorf("row scan error: [%w]", err)
		}
		bossMountMap[BossName(bossName)] = MountName(mountName)
	}
	return bossMountMap, nil
}
