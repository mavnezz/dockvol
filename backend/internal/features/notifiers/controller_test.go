package notifiers_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dockvol-backend/internal/features/notifiers"
	webhook_notifier "dockvol-backend/internal/features/notifiers/models/webhook"
	"dockvol-backend/internal/util/testutil"
)

func Test_SaveNotifier_Webhook_ListedThenDeleted(t *testing.T) {
	testutil.SetupDb(t)
	router := testutil.NewRouter()
	user := testutil.SignUpTestUser(t, router)
	workspace := testutil.CreateTestWorkspace(t, router, user.Token)

	created := notifiers.Notifier{
		WorkspaceID:  workspace.ID,
		Name:         "my-webhook",
		NotifierType: notifiers.NotifierTypeWebhook,
		WebhookNotifier: &webhook_notifier.WebhookNotifier{
			WebhookURL:    "https://example.com/hook",
			WebhookMethod: webhook_notifier.WebhookMethodPOST,
		},
	}
	var saved notifiers.Notifier
	testutil.MakePostAndUnmarshal(t, router, "/api/v1/notifiers", user.Token, created, http.StatusOK, &saved)
	assert.NotEqual(t, uuid.Nil, saved.ID)

	var listed []notifiers.Notifier
	testutil.MakeGetAndUnmarshal(
		t, router,
		"/api/v1/notifiers?workspace_id="+workspace.ID.String(),
		user.Token, http.StatusOK, &listed,
	)
	require.Len(t, listed, 1)
	assert.Equal(t, "my-webhook", listed[0].Name)

	testutil.MakeDelete(t, router, "/api/v1/notifiers/"+saved.ID.String(), user.Token, http.StatusOK)

	var afterDelete []notifiers.Notifier
	testutil.MakeGetAndUnmarshal(
		t, router,
		"/api/v1/notifiers?workspace_id="+workspace.ID.String(),
		user.Token, http.StatusOK, &afterDelete,
	)
	assert.Empty(t, afterDelete)
}
