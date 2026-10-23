//go:build !cloud

package main

import "github.com/bulga138/env-vault/utils"

func handleSyncSave() {
	utils.Warn("Cloud sync is not available in this build. Rebuild with -tags cloud to enable Firestore sync.")
}

func handleSyncLoad() {
	utils.Warn("Cloud sync is not available in this build. Rebuild with -tags cloud to enable Firestore sync.")
}