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

	"github.com/element-hq/dendrite/federationapi/storage/shared/receipt"
	"github.com/element-hq/dendrite/roomserver/types"
	"github.com/matrix-org/gomatrixserverlib/spec"
)

// AssociatePDUWithDestination creates an association that the
// destination queues will use to determine which JSON blobs to send
// to which servers.
func (d *Database) AssociatePDUWithDestinations(
	ctx context.Context,
	destinations map[spec.ServerName]struct{},
	dbReceipt *receipt.Receipt,
) error {
	return d.Writer.Do(d.DB, nil, func(txn *sql.Tx) error {
		var err error
		for destination := range destinations {
			err = d.FederationQueuePDUs.InsertQueuePDU(
				ctx,                // context
				txn,                // SQL transaction
				"",                 // transaction ID
				destination,        // destination server name
				dbReceipt.GetNID(), // NID from the federationapi_queue_json table
			)
		}
		return err
	})
}

// GetNextTransactionPDUs retrieves events from the database for
// the next pending transaction, up to the limit specified.
func (d *Database) GetPendingPDUs(
	ctx context.Context,
	serverName spec.ServerName,
	limit int,
) (
	events map[*receipt.Receipt]*types.HeaderedEvent,
	err error,
) {
	// Strictly speaking this doesn't need to be using the writer
	// since we are only performing selects, but since we don't have
	// a guarantee of transactional isolation, it's actually useful
	// to know in SQLite mode that nothing else is trying to modify
	// the database.
	events = make(map[*receipt.Receipt]*types.HeaderedEvent)
	err = d.Writer.Do(d.DB, nil, func(txn *sql.Tx) error {
		nids, err := d.FederationQueuePDUs.SelectQueuePDUs(ctx, txn, serverName, limit)
		if err != nil {
			return fmt.Errorf("SelectQueuePDUs: %w", err)
		}

		retrieve := make([]int64, 0, len(nids))
		for _, nid := range nids {
			if event, ok := d.Cache.GetFederationQueuedPDU(nid); ok {
				newReceipt := receipt.NewReceipt(nid)
				events[&newReceipt] = event
			} else {
				retrieve = append(retrieve, nid)
			}
		}

		blobs, err := d.FederationQueueJSON.SelectQueueJSON(ctx, txn, retrieve)
		if err != nil {
			return fmt.Errorf("SelectQueueJSON: %w", err)
		}

		for nid, blob := range blobs {
			var event types.HeaderedEvent
			if err := json.Unmarshal(blob, &event); err != nil {
				return fmt.Errorf("json.Unmarshal: %w", err)
			}
			newReceipt := receipt.NewReceipt(nid)
			events[&newReceipt] = &event
			d.Cache.StoreFederationQueuedPDU(nid, &event)
		}

		return nil
	})
	return
}

// CleanTransactionPDUs cleans up all associated events for a
// given transaction. This is done when the transaction was sent
// successfully.
func (d *Database) CleanPDUs(
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
		if err := d.FederationQueuePDUs.DeleteQueuePDUs(ctx, txn, serverName, nids); err != nil {
			return err
		}

		var deleteNIDs []int64
		for _, nid := range nids {
			count, err := d.FederationQueuePDUs.SelectQueuePDUReferenceJSONCount(ctx, txn, nid)
			if err != nil {
				return fmt.Errorf("SelectQueuePDUReferenceJSONCount: %w", err)
			}
			if count == 0 {
				deleteNIDs = append(deleteNIDs, nid)
				d.Cache.EvictFederationQueuedPDU(nid)
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

// GetPendingServerNames returns the server names that have PDUs
// waiting to be sent.
func (d *Database) GetPendingPDUServerNames(
	ctx context.Context,
) ([]spec.ServerName, error) {
	return d.FederationQueuePDUs.SelectQueuePDUServerNames(ctx, nil)
}
