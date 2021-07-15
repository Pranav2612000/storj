// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package satellitedb

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"storj.io/common/pb"
	"storj.io/common/storj"
	"storj.io/storj/satellite/internalpb"
	"storj.io/storj/satellite/overlay"
	"storj.io/storj/satellite/reputation"
	"storj.io/storj/satellite/satellitedb/dbx"
)

var _ reputation.DB = (*reputations)(nil)

type reputations struct {
	db *satelliteDB
}

func (reputations *reputations) Update(ctx context.Context, updateReq reputation.UpdateRequest, now time.Time) (_ *overlay.ReputationStatus, changed bool, err error) {
	defer mon.Task()(&ctx)(&err)

	nodeID := updateReq.NodeID

	var dbNode *dbx.Reputation
	var oldStatus overlay.ReputationStatus
	err = reputations.db.WithTx(ctx, func(ctx context.Context, tx *dbx.Tx) (err error) {
		_, err = tx.Tx.ExecContext(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE")
		if err != nil {
			return err
		}

		dbNode, err = tx.Get_Reputation_By_Id(ctx, dbx.Reputation_Id(nodeID.Bytes()))
		if errors.Is(err, sql.ErrNoRows) {
			historyBytes, err := pb.Marshal(&internalpb.AuditHistory{})
			if err != nil {
				return err
			}

			_, err = tx.Tx.ExecContext(ctx, `
				INSERT INTO reputations (id, audit_history)
				VALUES ($1, $2);
			`, nodeID.Bytes(), historyBytes)
			if err != nil {
				return err
			}

			dbNode, err = tx.Get_Reputation_By_Id(ctx, dbx.Reputation_Id(nodeID.Bytes()))
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		// do not update reputation if node is disqualified
		if dbNode.Disqualified != nil {
			return nil
		}

		oldStatus = overlay.ReputationStatus{
			Contained:             dbNode.Contained,
			Disqualified:          dbNode.Disqualified,
			UnknownAuditSuspended: dbNode.UnknownAuditSuspended,
			OfflineSuspended:      dbNode.OfflineSuspended,
			VettedAt:              dbNode.VettedAt,
		}

		isUp := updateReq.AuditOutcome != reputation.AuditOffline
		auditHistoryResponse, err := reputations.updateAuditHistoryWithTx(ctx, tx, nodeID, now, isUp, updateReq.AuditHistory)
		if err != nil {
			return err
		}

		updateFields := reputations.populateUpdateFields(dbNode, updateReq, auditHistoryResponse, now)
		dbNode, err = tx.Update_Reputation_By_Id(ctx, dbx.Reputation_Id(nodeID.Bytes()), updateFields)

		return err
	})
	if err != nil {
		return nil, false, Error.Wrap(err)
	}

	newStatus := overlay.ReputationStatus{
		Contained:             dbNode.Contained,
		Disqualified:          dbNode.Disqualified,
		UnknownAuditSuspended: dbNode.UnknownAuditSuspended,
		OfflineSuspended:      dbNode.OfflineSuspended,
		VettedAt:              dbNode.VettedAt,
	}

	return getNodeStatus(dbNode), !oldStatus.Equal(newStatus), nil
}

// SetNodeStatus updates node reputation status.
func (reputations *reputations) SetNodeStatus(ctx context.Context, id storj.NodeID, status overlay.ReputationStatus) (err error) {
	defer mon.Task()(&ctx)(&err)

	updateFields := dbx.Reputation_Update_Fields{
		Contained:             dbx.Reputation_Contained(status.Contained),
		Disqualified:          dbx.Reputation_Disqualified_Raw(status.Disqualified),
		UnknownAuditSuspended: dbx.Reputation_UnknownAuditSuspended_Raw(status.UnknownAuditSuspended),
		OfflineSuspended:      dbx.Reputation_OfflineSuspended_Raw(status.OfflineSuspended),
		VettedAt:              dbx.Reputation_VettedAt_Raw(status.VettedAt),
	}

	_, err = reputations.db.Update_Reputation_By_Id(ctx, dbx.Reputation_Id(id.Bytes()), updateFields)
	if err != nil {
		return Error.Wrap(err)
	}

	return nil

}
func (reputations *reputations) Get(ctx context.Context, nodeID storj.NodeID) (*reputation.Info, error) {
	res, err := reputations.db.Get_Reputation_By_Id(ctx, dbx.Reputation_Id(nodeID.Bytes()))
	if err != nil {
		if errs.Is(err, sql.ErrNoRows) {
			return nil, reputation.ErrNodeNotFound.New("no reputation entry for node")
		}
		return nil, Error.Wrap(err)
	}

	history, err := auditHistoryFromPB(res.AuditHistory)
	if err != nil {
		return nil, Error.Wrap(err)
	}

	return &reputation.Info{
		AuditSuccessCount:           res.AuditSuccessCount,
		TotalAuditCount:             res.TotalAuditCount,
		VettedAt:                    res.VettedAt,
		Contained:                   res.Contained,
		Disqualified:                res.Disqualified,
		Suspended:                   res.Suspended,
		UnknownAuditSuspended:       res.UnknownAuditSuspended,
		OfflineSuspended:            res.OfflineSuspended,
		UnderReview:                 res.UnderReview,
		OnlineScore:                 res.OnlineScore,
		AuditHistory:                *history,
		AuditReputationAlpha:        res.AuditReputationAlpha,
		AuditReputationBeta:         res.AuditReputationBeta,
		UnknownAuditReputationAlpha: res.UnknownAuditReputationAlpha,
		UnknownAuditReputationBeta:  res.UnknownAuditReputationBeta,
	}, nil
}

// DisqualifyNode disqualifies a storage node.
func (reputations *reputations) DisqualifyNode(ctx context.Context, nodeID storj.NodeID) (err error) {
	defer mon.Task()(&ctx)(&err)

	err = reputations.db.WithTx(ctx, func(ctx context.Context, tx *dbx.Tx) (err error) {
		_, err = tx.Tx.ExecContext(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE")
		if err != nil {
			return err
		}

		_, err = tx.Get_Reputation_By_Id(ctx, dbx.Reputation_Id(nodeID.Bytes()))
		if errors.Is(err, sql.ErrNoRows) {
			historyBytes, err := pb.Marshal(&internalpb.AuditHistory{})
			if err != nil {
				return err
			}

			_, err = tx.Tx.ExecContext(ctx, `
				INSERT INTO reputations (id, audit_history)
				VALUES ($1, $2);
			`, nodeID.Bytes(), historyBytes)
			if err != nil {
				return err
			}

		} else if err != nil {
			return err
		}

		updateFields := dbx.Reputation_Update_Fields{}
		updateFields.Disqualified = dbx.Reputation_Disqualified(time.Now().UTC())

		_, err = tx.Update_Reputation_By_Id(ctx, dbx.Reputation_Id(nodeID.Bytes()), updateFields)
		return err
	})
	return Error.Wrap(err)
}

// SuspendNodeUnknownAudit suspends a storage node for unknown audits.
func (reputations *reputations) SuspendNodeUnknownAudit(ctx context.Context, nodeID storj.NodeID, suspendedAt time.Time) (err error) {
	defer mon.Task()(&ctx)(&err)

	err = reputations.db.WithTx(ctx, func(ctx context.Context, tx *dbx.Tx) (err error) {
		_, err = tx.Tx.ExecContext(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE")
		if err != nil {
			return err
		}

		_, err = tx.Get_Reputation_By_Id(ctx, dbx.Reputation_Id(nodeID.Bytes()))
		if errors.Is(err, sql.ErrNoRows) {
			historyBytes, err := pb.Marshal(&internalpb.AuditHistory{})
			if err != nil {
				return err
			}

			_, err = tx.Tx.ExecContext(ctx, `
				INSERT INTO reputations (id, audit_history)
				VALUES ($1, $2);
			`, nodeID.Bytes(), historyBytes)
			if err != nil {
				return err
			}

		} else if err != nil {
			return err
		}

		updateFields := dbx.Reputation_Update_Fields{}
		updateFields.UnknownAuditSuspended = dbx.Reputation_UnknownAuditSuspended(suspendedAt.UTC())

		_, err = tx.Update_Reputation_By_Id(ctx, dbx.Reputation_Id(nodeID.Bytes()), updateFields)
		return err
	})
	return Error.Wrap(err)
}

// UnsuspendNodeUnknownAudit unsuspends a storage node for unknown audits.
func (reputations *reputations) UnsuspendNodeUnknownAudit(ctx context.Context, nodeID storj.NodeID) (err error) {
	defer mon.Task()(&ctx)(&err)

	err = reputations.db.WithTx(ctx, func(ctx context.Context, tx *dbx.Tx) (err error) {
		_, err = tx.Tx.ExecContext(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE")
		if err != nil {
			return err
		}

		_, err = tx.Get_Reputation_By_Id(ctx, dbx.Reputation_Id(nodeID.Bytes()))
		if errors.Is(err, sql.ErrNoRows) {
			historyBytes, err := pb.Marshal(&internalpb.AuditHistory{})
			if err != nil {
				return err
			}

			_, err = tx.Tx.ExecContext(ctx, `
				INSERT INTO reputations (id, audit_history)
				VALUES ($1, $2);
			`, nodeID.Bytes(), historyBytes)
			if err != nil {
				return err
			}

		} else if err != nil {
			return err
		}

		updateFields := dbx.Reputation_Update_Fields{}
		updateFields.UnknownAuditSuspended = dbx.Reputation_UnknownAuditSuspended_Null()

		_, err = tx.Update_Reputation_By_Id(ctx, dbx.Reputation_Id(nodeID.Bytes()), updateFields)
		return err

	})
	return Error.Wrap(err)
}

// Init creates an entry for a node with its reputaion in zero value.
func (reputations *reputations) Init(ctx context.Context, nodeID storj.NodeID) error {
	historyBytes, err := pb.Marshal(&internalpb.AuditHistory{})
	if err != nil {
		return err
	}

	_, err = reputations.db.Create_Reputation(ctx, dbx.Reputation_Id(nodeID.Bytes()), dbx.Reputation_AuditHistory(historyBytes), dbx.Reputation_Create_Fields{})
	return Error.Wrap(err)
}

func (reputations *reputations) populateUpdateFields(dbNode *dbx.Reputation, updateReq reputation.UpdateRequest, auditHistoryResponse *reputation.UpdateAuditHistoryResponse, now time.Time) dbx.Reputation_Update_Fields {

	update := reputations.populateUpdateNodeStats(dbNode, updateReq, auditHistoryResponse, now)
	updateFields := dbx.Reputation_Update_Fields{}
	if update.VettedAt.set {
		updateFields.VettedAt = dbx.Reputation_VettedAt(update.VettedAt.value)
	}
	if update.TotalAuditCount.set {
		updateFields.TotalAuditCount = dbx.Reputation_TotalAuditCount(update.TotalAuditCount.value)
	}
	if update.AuditReputationAlpha.set {
		updateFields.AuditReputationAlpha = dbx.Reputation_AuditReputationAlpha(update.AuditReputationAlpha.value)
	}
	if update.AuditReputationBeta.set {
		updateFields.AuditReputationBeta = dbx.Reputation_AuditReputationBeta(update.AuditReputationBeta.value)
	}
	if update.Disqualified.set {
		updateFields.Disqualified = dbx.Reputation_Disqualified(update.Disqualified.value)
	}
	if update.UnknownAuditReputationAlpha.set {
		updateFields.UnknownAuditReputationAlpha = dbx.Reputation_UnknownAuditReputationAlpha(update.UnknownAuditReputationAlpha.value)
	}
	if update.UnknownAuditReputationBeta.set {
		updateFields.UnknownAuditReputationBeta = dbx.Reputation_UnknownAuditReputationBeta(update.UnknownAuditReputationBeta.value)
	}
	if update.UnknownAuditSuspended.set {
		if update.UnknownAuditSuspended.isNil {
			updateFields.UnknownAuditSuspended = dbx.Reputation_UnknownAuditSuspended_Null()
		} else {
			updateFields.UnknownAuditSuspended = dbx.Reputation_UnknownAuditSuspended(update.UnknownAuditSuspended.value)
		}
	}
	if update.AuditSuccessCount.set {
		updateFields.AuditSuccessCount = dbx.Reputation_AuditSuccessCount(update.AuditSuccessCount.value)
	}
	if update.Contained.set {
		updateFields.Contained = dbx.Reputation_Contained(update.Contained.value)
	}
	if updateReq.AuditOutcome == reputation.AuditSuccess {
		updateFields.AuditSuccessCount = dbx.Reputation_AuditSuccessCount(dbNode.AuditSuccessCount + 1)
	}

	if update.OnlineScore.set {
		updateFields.OnlineScore = dbx.Reputation_OnlineScore(update.OnlineScore.value)
	}
	if update.OfflineSuspended.set {
		if update.OfflineSuspended.isNil {
			updateFields.OfflineSuspended = dbx.Reputation_OfflineSuspended_Null()
		} else {
			updateFields.OfflineSuspended = dbx.Reputation_OfflineSuspended(update.OfflineSuspended.value)
		}
	}
	if update.OfflineUnderReview.set {
		if update.OfflineUnderReview.isNil {
			updateFields.UnderReview = dbx.Reputation_UnderReview_Null()
		} else {
			updateFields.UnderReview = dbx.Reputation_UnderReview(update.OfflineUnderReview.value)
		}
	}

	return updateFields
}

func (reputations *reputations) populateUpdateNodeStats(dbNode *dbx.Reputation, updateReq reputation.UpdateRequest, auditHistoryResponse *reputation.UpdateAuditHistoryResponse, now time.Time) updateNodeStats {
	// there are three audit outcomes: success, failure, and unknown
	// if a node fails enough audits, it gets disqualified
	// if a node gets enough "unknown" audits, it gets put into suspension
	// if a node gets enough successful audits, and is in suspension, it gets removed from suspension
	auditAlpha := dbNode.AuditReputationAlpha
	auditBeta := dbNode.AuditReputationBeta
	unknownAuditAlpha := dbNode.UnknownAuditReputationAlpha
	unknownAuditBeta := dbNode.UnknownAuditReputationBeta
	totalAuditCount := dbNode.TotalAuditCount
	vettedAt := dbNode.VettedAt

	var updatedTotalAuditCount int64

	switch updateReq.AuditOutcome {
	case reputation.AuditSuccess:
		// for a successful audit, increase reputation for normal *and* unknown audits
		auditAlpha, auditBeta, updatedTotalAuditCount = updateReputation(
			true,
			auditAlpha,
			auditBeta,
			updateReq.AuditLambda,
			updateReq.AuditWeight,
			totalAuditCount,
		)
		// we will use updatedTotalAuditCount from the updateReputation call above
		unknownAuditAlpha, unknownAuditBeta, _ = updateReputation(
			true,
			unknownAuditAlpha,
			unknownAuditBeta,
			updateReq.AuditLambda,
			updateReq.AuditWeight,
			totalAuditCount,
		)
	case reputation.AuditFailure:
		// for audit failure, only update normal alpha/beta
		auditAlpha, auditBeta, updatedTotalAuditCount = updateReputation(
			false,
			auditAlpha,
			auditBeta,
			updateReq.AuditLambda,
			updateReq.AuditWeight,
			totalAuditCount,
		)
	case reputation.AuditUnknown:
		// for audit unknown, only update unknown alpha/beta
		unknownAuditAlpha, unknownAuditBeta, updatedTotalAuditCount = updateReputation(
			false,
			unknownAuditAlpha,
			unknownAuditBeta,
			updateReq.AuditLambda,
			updateReq.AuditWeight,
			totalAuditCount,
		)
	case reputation.AuditOffline:
		// for audit offline, only update total audit count
		updatedTotalAuditCount = totalAuditCount + 1
	}

	mon.FloatVal("audit_reputation_alpha").Observe(auditAlpha)                //mon:locked
	mon.FloatVal("audit_reputation_beta").Observe(auditBeta)                  //mon:locked
	mon.FloatVal("unknown_audit_reputation_alpha").Observe(unknownAuditAlpha) //mon:locked
	mon.FloatVal("unknown_audit_reputation_beta").Observe(unknownAuditBeta)   //mon:locked
	mon.FloatVal("audit_online_score").Observe(auditHistoryResponse.NewScore) //mon:locked

	isUp := updateReq.AuditOutcome != reputation.AuditOffline

	updateFields := updateNodeStats{
		NodeID:                      updateReq.NodeID,
		TotalAuditCount:             int64Field{set: true, value: updatedTotalAuditCount},
		AuditReputationAlpha:        float64Field{set: true, value: auditAlpha},
		AuditReputationBeta:         float64Field{set: true, value: auditBeta},
		UnknownAuditReputationAlpha: float64Field{set: true, value: unknownAuditAlpha},
		UnknownAuditReputationBeta:  float64Field{set: true, value: unknownAuditBeta},
		// Updating node stats always exits it from containment mode
		Contained: boolField{set: true, value: false},
		// always update online score
		OnlineScore: float64Field{set: true, value: auditHistoryResponse.NewScore},
	}

	if vettedAt == nil && updatedTotalAuditCount >= updateReq.AuditsRequiredForVetting {
		updateFields.VettedAt = timeField{set: true, value: now}
	}

	// disqualification case a
	//   a) Success/fail audit reputation falls below audit DQ threshold
	auditRep := auditAlpha / (auditAlpha + auditBeta)
	if auditRep <= updateReq.AuditDQ {
		reputations.db.log.Info("Disqualified", zap.String("DQ type", "audit failure"), zap.String("Node ID", updateReq.NodeID.String()))
		mon.Meter("bad_audit_dqs").Mark(1) //mon:locked
		updateFields.Disqualified = timeField{set: true, value: now}
	}

	// if unknown audit rep goes below threshold, suspend node. Otherwise unsuspend node.
	unknownAuditRep := unknownAuditAlpha / (unknownAuditAlpha + unknownAuditBeta)
	if unknownAuditRep <= updateReq.AuditDQ {
		if dbNode.UnknownAuditSuspended == nil {
			reputations.db.log.Info("Suspended", zap.String("Node ID", updateFields.NodeID.String()), zap.String("Category", "Unknown Audits"))
			updateFields.UnknownAuditSuspended = timeField{set: true, value: now}
		}

		// disqualification case b
		//   b) Node is suspended (success/unknown reputation below audit DQ threshold)
		//        AND the suspended grace period has elapsed
		//        AND audit outcome is unknown or failed

		// if suspended grace period has elapsed and audit outcome was failed or unknown,
		// disqualify node. Set suspended to nil if node is disqualified
		// NOTE: if updateFields.Suspended is set, we just suspended the node so it will not be disqualified
		if updateReq.AuditOutcome != reputation.AuditSuccess {
			if dbNode.UnknownAuditSuspended != nil && !updateFields.UnknownAuditSuspended.set &&
				time.Since(*dbNode.UnknownAuditSuspended) > updateReq.SuspensionGracePeriod &&
				updateReq.SuspensionDQEnabled {
				reputations.db.log.Info("Disqualified", zap.String("DQ type", "suspension grace period expired for unknown audits"), zap.String("Node ID", updateReq.NodeID.String()))
				mon.Meter("unknown_suspension_dqs").Mark(1) //mon:locked
				updateFields.Disqualified = timeField{set: true, value: now}
				updateFields.UnknownAuditSuspended = timeField{set: true, isNil: true}
			}
		}
	} else if dbNode.UnknownAuditSuspended != nil {
		reputations.db.log.Info("Suspension lifted", zap.String("Category", "Unknown Audits"), zap.String("Node ID", updateFields.NodeID.String()))
		updateFields.UnknownAuditSuspended = timeField{set: true, isNil: true}
	}

	if isUp {
		updateFields.LastContactSuccess = timeField{set: true, value: now}
	} else {
		updateFields.LastContactFailure = timeField{set: true, value: now}
	}

	if updateReq.AuditOutcome == reputation.AuditSuccess {
		updateFields.AuditSuccessCount = int64Field{set: true, value: dbNode.AuditSuccessCount + 1}
	}

	// if suspension not enabled, skip penalization and unsuspend node if applicable
	if !updateReq.AuditHistory.OfflineSuspensionEnabled {
		if dbNode.OfflineSuspended != nil {
			updateFields.OfflineSuspended = timeField{set: true, isNil: true}
		}
		if dbNode.UnderReview != nil {
			updateFields.OfflineUnderReview = timeField{set: true, isNil: true}
		}
		return updateFields
	}

	// only penalize node if online score is below threshold and
	// if it has enough completed windows to fill a tracking period
	penalizeOfflineNode := false
	if auditHistoryResponse.NewScore < updateReq.AuditHistory.OfflineThreshold && auditHistoryResponse.TrackingPeriodFull {
		penalizeOfflineNode = true
	}

	// Suspension and disqualification for offline nodes
	if dbNode.UnderReview != nil {
		// move node in and out of suspension as needed during review period
		if !penalizeOfflineNode && dbNode.OfflineSuspended != nil {
			updateFields.OfflineSuspended = timeField{set: true, isNil: true}
		} else if penalizeOfflineNode && dbNode.OfflineSuspended == nil {
			updateFields.OfflineSuspended = timeField{set: true, value: now}
		}

		gracePeriodEnd := dbNode.UnderReview.Add(updateReq.AuditHistory.GracePeriod)
		trackingPeriodEnd := gracePeriodEnd.Add(updateReq.AuditHistory.TrackingPeriod)
		trackingPeriodPassed := now.After(trackingPeriodEnd)

		// after tracking period has elapsed, if score is good, clear under review
		// otherwise, disqualify node (if OfflineDQEnabled feature flag is true)
		if trackingPeriodPassed {
			if penalizeOfflineNode {
				if updateReq.AuditHistory.OfflineDQEnabled {
					reputations.db.log.Info("Disqualified", zap.String("DQ type", "node offline"), zap.String("Node ID", updateReq.NodeID.String()))
					mon.Meter("offline_dqs").Mark(1) //mon:locked
					updateFields.Disqualified = timeField{set: true, value: now}
				}
			} else {
				updateFields.OfflineUnderReview = timeField{set: true, isNil: true}
				updateFields.OfflineSuspended = timeField{set: true, isNil: true}
			}
		}
	} else if penalizeOfflineNode {
		// suspend node for being offline and begin review period
		updateFields.OfflineUnderReview = timeField{set: true, value: now}
		updateFields.OfflineSuspended = timeField{set: true, value: now}
	}

	return updateFields
}

type int64Field struct {
	set   bool
	value int64
}

type float64Field struct {
	set   bool
	value float64
}

type boolField struct {
	set   bool
	value bool
}

type timeField struct {
	set   bool
	isNil bool
	value time.Time
}

type updateNodeStats struct {
	NodeID                      storj.NodeID
	VettedAt                    timeField
	TotalAuditCount             int64Field
	AuditReputationAlpha        float64Field
	AuditReputationBeta         float64Field
	Disqualified                timeField
	UnknownAuditReputationAlpha float64Field
	UnknownAuditReputationBeta  float64Field
	UnknownAuditSuspended       timeField
	LastContactSuccess          timeField
	LastContactFailure          timeField
	AuditSuccessCount           int64Field
	Contained                   boolField
	OfflineUnderReview          timeField
	OfflineSuspended            timeField
	OnlineScore                 float64Field
}

func getNodeStatus(dbNode *dbx.Reputation) *overlay.ReputationStatus {
	return &overlay.ReputationStatus{
		Contained:             dbNode.Contained,
		VettedAt:              dbNode.VettedAt,
		Disqualified:          dbNode.Disqualified,
		UnknownAuditSuspended: dbNode.UnknownAuditSuspended,
		OfflineSuspended:      dbNode.OfflineSuspended,
	}

}

// updateReputation uses the Beta distribution model to determine a node's reputation.
// lambda is the "forgetting factor" which determines how much past info is kept when determining current reputation score.
// w is the normalization weight that affects how severely new updates affect the current reputation distribution.
func updateReputation(isSuccess bool, alpha, beta, lambda, w float64, totalCount int64) (newAlpha, newBeta float64, updatedCount int64) {
	// v is a single feedback value that allows us to update both alpha and beta
	var v float64 = -1
	if isSuccess {
		v = 1
	}
	newAlpha = lambda*alpha + w*(1+v)/2
	newBeta = lambda*beta + w*(1-v)/2
	return newAlpha, newBeta, totalCount + 1
}