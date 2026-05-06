package operation_log_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oplog "circular-board/internal/operation_log"
)

// ═══════════════════════════════════════════════════════════
// Repository.Save（ログ保存）
// ═══════════════════════════════════════════════════════════

// TestOperationLogRepository_Save_PermissionUpdate
// 権限変更のログが保存される
func TestOperationLogRepository_Save_PermissionUpdate(t *testing.T) {
	operatorID := insertTestUser(t, "SA_OPLOG1", "sa-oplog1@oplog.test", "system_admin")
	targetID := "role-id:feature-id"

	repo := oplog.NewRepository(testPool)
	log, err := repo.Save(context.Background(), oplog.SaveInput{
		OperatorID:    operatorID,
		OperationType: "permission_update",
		TargetType:    "role_permission",
		TargetID:      &targetID,
		BeforeValue:   strPtr(`{"can_view":true,"can_create":false}`),
		AfterValue:    strPtr(`{"can_view":true,"can_create":true}`),
	})
	require.NoError(t, err)
	require.NotNil(t, log)

	assert.NotEqual(t, uuid.Nil, log.ID)
	assert.Equal(t, operatorID, log.OperatorID)
	assert.Equal(t, "permission_update", log.OperationType)
	assert.Equal(t, "role_permission", log.TargetType)
	assert.Equal(t, &targetID, log.TargetID)

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM operation_logs WHERE id = $1`, log.ID)
	})
}

// TestOperationLogRepository_Save_EmergencyAppointment
// 緊急任命のログが保存される
func TestOperationLogRepository_Save_EmergencyAppointment(t *testing.T) {
	operatorID := insertTestUser(t, "SA_OPLOG2", "sa-oplog2@oplog.test", "system_admin")
	targetUserIDStr := uuid.New().String()

	repo := oplog.NewRepository(testPool)
	log, err := repo.Save(context.Background(), oplog.SaveInput{
		OperatorID:    operatorID,
		OperationType: "emergency_appointment",
		TargetType:    "user",
		TargetID:      &targetUserIDStr,
		BeforeValue:   strPtr(`{"role":"user"}`),
		AfterValue:    strPtr(`{"role":"vice_admin"}`),
	})
	require.NoError(t, err)
	require.NotNil(t, log)

	assert.Equal(t, "emergency_appointment", log.OperationType)
	assert.Equal(t, "user", log.TargetType)

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM operation_logs WHERE id = $1`, log.ID)
	})
}

// TestOperationLogRepository_Save_BeforeAfterValueJSON
// 変更前・変更後の値がJSONで保存される
func TestOperationLogRepository_Save_BeforeAfterValueJSON(t *testing.T) {
	operatorID := insertTestUser(t, "SA_OPLOG3", "sa-oplog3@oplog.test", "system_admin")
	targetID := "target-123"
	beforeJSON := `{"can_view":false,"can_create":false,"can_edit":false,"can_delete":false}`
	afterJSON := `{"can_view":true,"can_create":true,"can_edit":false,"can_delete":false}`

	repo := oplog.NewRepository(testPool)
	log, err := repo.Save(context.Background(), oplog.SaveInput{
		OperatorID:    operatorID,
		OperationType: "permission_update",
		TargetType:    "role_permission",
		TargetID:      &targetID,
		BeforeValue:   &beforeJSON,
		AfterValue:    &afterJSON,
	})
	require.NoError(t, err)

	// DBから直接取得して確認
	var savedBefore, savedAfter string
	err = testPool.QueryRow(context.Background(),
		`SELECT before_value::text, after_value::text FROM operation_logs WHERE id = $1`, log.ID,
	).Scan(&savedBefore, &savedAfter)
	require.NoError(t, err)
	assert.Contains(t, savedBefore, "can_view", "before_valueにJSONが保存されているべき")
	assert.Contains(t, savedAfter, "can_create", "after_valueにJSONが保存されているべき")

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM operation_logs WHERE id = $1`, log.ID)
	})
}

// TestOperationLogRepository_Save_OperatorIDCorrect
// 操作者IDが正しく保存される
func TestOperationLogRepository_Save_OperatorIDCorrect(t *testing.T) {
	operatorID := insertTestUser(t, "SA_OPLOG4", "sa-oplog4@oplog.test", "system_admin")

	repo := oplog.NewRepository(testPool)
	log, err := repo.Save(context.Background(), oplog.SaveInput{
		OperatorID:    operatorID,
		OperationType: "permission_update",
		TargetType:    "role_permission",
	})
	require.NoError(t, err)
	assert.Equal(t, operatorID, log.OperatorID, "操作者IDが正しく保存されているべき")

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM operation_logs WHERE id = $1`, log.ID)
	})
}

// TestOperationLogRepository_Save_CreatedAtSet
// 操作日時が正しく保存される
func TestOperationLogRepository_Save_CreatedAtSet(t *testing.T) {
	operatorID := insertTestUser(t, "SA_OPLOG5", "sa-oplog5@oplog.test", "system_admin")
	before := time.Now().UTC().Add(-time.Second)

	repo := oplog.NewRepository(testPool)
	log, err := repo.Save(context.Background(), oplog.SaveInput{
		OperatorID:    operatorID,
		OperationType: "permission_update",
		TargetType:    "role_permission",
	})
	require.NoError(t, err)

	after := time.Now().UTC().Add(time.Second)
	assert.True(t, log.CreatedAt.After(before) && log.CreatedAt.Before(after),
		"created_atが現在時刻付近に設定されているべき")

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM operation_logs WHERE id = $1`, log.ID)
	})
}

// TestOperationLogRepository_Save_NullBeforeAfter
// before_value/after_valueがNULLでも保存できる
func TestOperationLogRepository_Save_NullBeforeAfter(t *testing.T) {
	operatorID := insertTestUser(t, "SA_OPLOG6", "sa-oplog6@oplog.test", "system_admin")

	repo := oplog.NewRepository(testPool)
	log, err := repo.Save(context.Background(), oplog.SaveInput{
		OperatorID:    operatorID,
		OperationType: "permission_update",
		TargetType:    "role_permission",
		BeforeValue:   nil,
		AfterValue:    nil,
	})
	require.NoError(t, err)
	assert.Nil(t, log.BeforeValue)
	assert.Nil(t, log.AfterValue)

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM operation_logs WHERE id = $1`, log.ID)
	})
}

// ═══════════════════════════════════════════════════════════
// Repository.List（ログ取得）
// ═══════════════════════════════════════════════════════════

// TestOperationLogRepository_List_DescendingOrder
// 日時の降順で返る
func TestOperationLogRepository_List_DescendingOrder(t *testing.T) {
	operatorID := insertTestUser(t, "SA_LIST1", "sa-list1@oplog.test", "system_admin")

	repo := oplog.NewRepository(testPool)

	// 3件保存
	ids := make([]interface{}, 0, 3)
	for i := range 3 {
		log, err := repo.Save(context.Background(), oplog.SaveInput{
			OperatorID:    operatorID,
			OperationType: "permission_update",
			TargetType:    "role_permission",
			TargetID:      strPtr("order-test-" + string(rune('A'+i))),
		})
		require.NoError(t, err)
		ids = append(ids, log.ID)
		// 時刻差をつける
		time.Sleep(5 * time.Millisecond)
	}
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = testPool.Exec(context.Background(), `DELETE FROM operation_logs WHERE id = $1`, id)
		}
	})

	logs, err := repo.List(context.Background())
	require.NoError(t, err)

	// ログが存在し、降順になっているかチェック
	for i := 1; i < len(logs); i++ {
		assert.True(t, !logs[i-1].CreatedAt.Before(logs[i].CreatedAt),
			"ログは降順（新しい順）であるべき")
	}
}

// TestOperationLogRepository_List_Empty
// ログがない場合は空スライスが返る
func TestOperationLogRepository_List_Empty(t *testing.T) {
	repo := oplog.NewRepository(testPool)
	// 全件消した後に確認することはしない（他テストに影響するため）
	// 代わりに空スライスが返ることだけを確認
	logs, err := repo.List(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, logs)
}

// ═══════════════════════════════════════════════════════════
// Repository.ListByOperationType（操作種別での絞り込み）
// ═══════════════════════════════════════════════════════════

// TestOperationLogRepository_ListByOperationType_FilterCorrect
// 操作種別で絞り込める
func TestOperationLogRepository_ListByOperationType_FilterCorrect(t *testing.T) {
	operatorID := insertTestUser(t, "SA_FILTER1", "sa-filter1@oplog.test", "system_admin")

	repo := oplog.NewRepository(testPool)
	log1, err := repo.Save(context.Background(), oplog.SaveInput{
		OperatorID:    operatorID,
		OperationType: "permission_update",
		TargetType:    "role_permission",
	})
	require.NoError(t, err)

	log2, err := repo.Save(context.Background(), oplog.SaveInput{
		OperatorID:    operatorID,
		OperationType: "emergency_appointment",
		TargetType:    "user",
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM operation_logs WHERE id = $1`, log1.ID)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM operation_logs WHERE id = $1`, log2.ID)
	})

	// permission_updateのみ取得
	permLogs, err := repo.ListByOperationType(context.Background(), "permission_update")
	require.NoError(t, err)
	for _, l := range permLogs {
		assert.Equal(t, "permission_update", l.OperationType,
			"permission_update以外が含まれてはならない")
	}
	// log1が含まれる
	found := false
	for _, l := range permLogs {
		if l.ID == log1.ID {
			found = true
		}
	}
	assert.True(t, found, "permission_updateのログが含まれるべき")

	// emergency_appointmentのみ取得
	eaLogs, err := repo.ListByOperationType(context.Background(), "emergency_appointment")
	require.NoError(t, err)
	for _, l := range eaLogs {
		assert.Equal(t, "emergency_appointment", l.OperationType)
	}
	found = false
	for _, l := range eaLogs {
		if l.ID == log2.ID {
			found = true
		}
	}
	assert.True(t, found, "emergency_appointmentのログが含まれるべき")
}

// ═══════════════════════════════════════════════════════════
// Repository.ListByOperatorID（操作者IDでの絞り込み）
// ═══════════════════════════════════════════════════════════

// TestOperationLogRepository_ListByOperatorID_FilterCorrect
// 操作者IDで絞り込める
func TestOperationLogRepository_ListByOperatorID_FilterCorrect(t *testing.T) {
	operatorA := insertTestUser(t, "SA_OA", "sa-oa@oplog.test", "system_admin")
	operatorB := insertTestUser(t, "SA_OB", "sa-ob@oplog.test", "system_admin")

	repo := oplog.NewRepository(testPool)
	logA, err := repo.Save(context.Background(), oplog.SaveInput{
		OperatorID:    operatorA,
		OperationType: "permission_update",
		TargetType:    "role_permission",
	})
	require.NoError(t, err)

	logB, err := repo.Save(context.Background(), oplog.SaveInput{
		OperatorID:    operatorB,
		OperationType: "permission_update",
		TargetType:    "role_permission",
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM operation_logs WHERE id = $1`, logA.ID)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM operation_logs WHERE id = $1`, logB.ID)
	})

	// operatorAのログのみ取得
	logsA, err := repo.ListByOperatorID(context.Background(), operatorA)
	require.NoError(t, err)

	for _, l := range logsA {
		assert.Equal(t, operatorA, l.OperatorID,
			"operatorAのIDで絞り込んだ結果にoperatorBのログが含まれてはならない")
	}

	foundA := false
	for _, l := range logsA {
		if l.ID == logA.ID {
			foundA = true
		}
	}
	assert.True(t, foundA, "operatorAのログが含まれるべき")
}

// TestOperationLogRepository_ListByOperatorID_DescendingOrder
// 日時の降順で返る
func TestOperationLogRepository_ListByOperatorID_DescendingOrder(t *testing.T) {
	operatorID := insertTestUser(t, "SA_ORD2", "sa-ord2@oplog.test", "system_admin")

	repo := oplog.NewRepository(testPool)
	ids := make([]uuid.UUID, 0, 3)
	for range 3 {
		log, err := repo.Save(context.Background(), oplog.SaveInput{
			OperatorID:    operatorID,
			OperationType: "permission_update",
			TargetType:    "role_permission",
		})
		require.NoError(t, err)
		ids = append(ids, log.ID)
		time.Sleep(5 * time.Millisecond)
	}
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = testPool.Exec(context.Background(), `DELETE FROM operation_logs WHERE id = $1`, id)
		}
	})

	logs, err := repo.ListByOperatorID(context.Background(), operatorID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(logs), 3)

	for i := 1; i < len(logs); i++ {
		assert.True(t, !logs[i-1].CreatedAt.Before(logs[i].CreatedAt),
			"ログは降順（新しい順）であるべき")
	}
}
