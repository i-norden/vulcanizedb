// Copyright 2018 Vulcanize
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vat_init

import (
	"fmt"
	"github.com/vulcanize/vulcanizedb/pkg/datastore/postgres"
	"github.com/vulcanize/vulcanizedb/pkg/transformers/shared"
	"github.com/vulcanize/vulcanizedb/pkg/transformers/shared/constants"
	"log"
)

type VatInitRepository struct {
	db *postgres.DB
}

func (repository VatInitRepository) Create(headerID int64, models []interface{}) error {
	tx, dBaseErr := repository.db.Begin()
	if dBaseErr != nil {
		return dBaseErr
	}

	for _, model := range models {
		vatInit, ok := model.(VatInitModel)
		if !ok {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Println("failed to rollback ", rollbackErr)
			}
			return fmt.Errorf("model of type %T, not %T", model, VatInitModel{})
		}

		_, execErr := tx.Exec(
			`INSERT INTO maker.vat_init (header_id, ilk, log_idx, tx_idx, raw_log)
			VALUES($1, $2, $3, $4, $5)`,
			headerID, vatInit.Ilk, vatInit.LogIndex, vatInit.TransactionIndex, vatInit.Raw,
		)
		if execErr != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Println("failed to rollback ", rollbackErr)
			}
			return execErr
		}
	}

	checkHeaderErr := shared.MarkHeaderCheckedInTransaction(headerID, tx, constants.VatInitChecked)
	if checkHeaderErr != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			log.Println("failed to rollback ", rollbackErr)
		}
		return checkHeaderErr
	}

	return tx.Commit()
}

func (repository VatInitRepository) MarkHeaderChecked(headerID int64) error {
	return shared.MarkHeaderChecked(headerID, repository.db, constants.VatInitChecked)
}

func (repository *VatInitRepository) SetDB(db *postgres.DB) {
	repository.db = db
}
