// Copyright 2024 New Vector Ltd.
// Copyright 2020 The Matrix.org Foundation C.I.C.
//
// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-Element-Commercial
// Please see LICENSE files in the repository root for full details.

package shared

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/element-hq/dendrite/federationapi/storage/shared/receipt"
	"github.com/matrix-org/gomatrixserverlib"
	"github.com/matrix-org/gomatrixserverlib/spec"
)

// defaultExpiry for EDUs if not listed below
var defaultExpiry = time.Hour * 24

// defaultExpireEDUTypes contains EDUs which can/should be expired after a given time
// if the target server isn't reachable for some reason.
var defaultExpireEDUTypes = map[string]time.Duration{
	spec.MTyping:   time.Minute,
	spec.MPresence: time.Minute * 10,
}

// AssociateEDUWithDestination creates an association that the
// destination queues will use to determine which JSON blobs to send
// to which servers.
func (d *Database) AssociateEDUWithDestinations(
	ctx context.Context,
	destinations map[spec.ServerName]struct{},
	dbReceipt *receipt.Receipt,
	eduType string,
	expireEDUTypes map[string]time.Duration,
) error {
	if expireEDUTypes == nil {
		expireEDUTypes = defaultExpireEDUTypes
	}
	expiresAt := spec.AsTimestamp(time.Now().Add(defaultExpiry))
	if duration, ok := expireEDUTypes[eduType]; ok {
		// Keep EDUs for at least x minutes before deleting them
		expiresAt = spec.AsTimestamp(time.Now().Add(duration))
	}
	// We forcibly set m.direct_to_device and m.device_list_update events
	// to 0, as we always want them to be delivered. (required for E2EE)
	if eduType == spec.MDirectToDevice || eduType == spec.MDeviceListUpdate {
		expiresAt = 0
	}
	return d.Writer.Do(d.DB, nil, func(txn *sql.Tx) error {
		var err error
		for destination := range destinations {
			err = d.FederationQueueEDUs.InsertQueueEDU(
				ctx,                // context
				txn,                // SQL transaction
				eduType,            // EDU type for coalescing
				destination,        // destination server name
				dbReceipt.GetNID(), // NID from the federationapi_queue_json table
				expiresAt,          // The timestamp this EDU will expire
			)
		}
		return err
	})
}

// GetNextTransactionEDUs retrieves events from the database for
// the next pending transaction, up to the limit specified.
func (d *Database) GetPendingEDUs(
	ctx context.Context,
	serverName spec.ServerName,
	limit int,
) (
	edus map[*receipt.Receipt]*gomatrixserverlib.EDU,
	err error,
) {
	edus = make(map[*receipt.Receipt]*gomatrixserverlib.EDU)
	err = d.Writer.Do(d.DB, nil, func(txn *sql.Tx) error {
		nids, err := d.FederationQueueEDUs.SelectQueueEDUs(ctx, txn, serverName, limit)
		if err != nil {
			return fmt.Errorf("SelectQueueEDUs: %w", err)
		}

		retrieve := make([]int64, 0, len(nids))
		for _, nid := range nids {
			if edu, ok := d.Cache.GetFederationQueuedEDU(nid); ok {
				newReceipt := receipt.NewReceipt(nid)
				edus[&newReceipt] = edu
			} else {
				retrieve = append(retrieve, nid)
			}
		}

		blobs, err := d.FederationQueueJSON.SelectQueueJSON(ctx, txn, retrieve)
		if err != nil {
			return fmt.Errorf("SelectQueueJSON: %w", err)
		}

		for nid, blob := range blobs {
			var event gomatrixserverlib.EDU
			if err := json.Unmarshal(blob, &event); err != nil {
				return fmt.Errorf("json.Unmarshal: %w", err)
			}
			newReceipt := receipt.NewReceipt(nid)
			edus[&newReceipt] = &event
			d.Cache.StoreFederationQueuedEDU(nid, &event)
		}

		return nil
	})
	return
}

// CleanEDUs cleans up all specified EDUs. This is done when a
// transaction was sent successfully.
func (d *Database) CleanEDUs(
	ctx context.Context,
	serverName spec.ServerName,
	receipts []*receipt.Receipt,
) error {
	if len(receipts) == 0 {
		return errors.New("expected receipt")
	}

	nids := make([]int64, len(receipts))
	for i := range receipts {
		nids[i] = receipts[i].GetNID()
	}

	return d.Writer.Do(d.DB, nil, func(txn *sql.Tx) error {
		if err := d.FederationQueueEDUs.DeleteQueueEDUs(ctx, txn, serverName, nids); err != nil {
			return err
		}

		var deleteNIDs []int64
		for _, nid := range nids {
			count, err := d.FederationQueueEDUs.SelectQueueEDUReferenceJSONCount(ctx, txn, nid)
			if err != nil {
				return fmt.Errorf("SelectQueueEDUReferenceJSONCount: %w", err)
			}
			if count == 0 {
				deleteNIDs = append(deleteNIDs, nid)
				d.Cache.EvictFederationQueuedEDU(nid)
			}
		}

		if len(deleteNIDs) > 0 {
			if err := d.FederationQueueJSON.DeleteQueueJSON(ctx, txn, deleteNIDs); err != nil {
				return fmt.Errorf("DeleteQueueJSON: %w", err)
			}
		}

		return nil
	})
}

// GetPendingServerNames returns the server names that have EDUs
// waiting to be sent.
func (d *Database) GetPendingEDUServerNames(
	ctx context.Context,
) ([]spec.ServerName, error) {
	return d.FederationQueueEDUs.SelectQueueEDUServerNames(ctx, nil)
}

// DeleteExpiredEDUs deletes expired EDUs and evicts them from the cache.
func (d *Database) DeleteExpiredEDUs(ctx context.Context) error {
	var jsonNIDs []int64
	err := d.Writer.Do(d.DB, nil, func(txn *sql.Tx) (err error) {
		expiredBefore := spec.AsTimestamp(time.Now())
		jsonNIDs, err = d.FederationQueueEDUs.SelectExpiredEDUs(ctx, txn, expiredBefore)
		if err != nil {
			return err
		}
		if len(jsonNIDs) == 0 {
			return nil
		}

		if err = d.FederationQueueJSON.DeleteQueueJSON(ctx, txn, jsonNIDs); err != nil {
			return err
		}

		return d.FederationQueueEDUs.DeleteExpiredEDUs(ctx, txn, expiredBefore)
	})

	if err != nil {
		return err
	}

	for i := range jsonNIDs {
		d.Cache.EvictFederationQueuedEDU(jsonNIDs[i])
	}

	return nil
}
