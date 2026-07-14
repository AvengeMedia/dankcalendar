package ipc

import (
	"context"
	"strings"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/internal/accounts"
	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
	"github.com/AvengeMedia/dankgo/log"
)

func handleMicrosoftStart(_ context.Context, w *ConnWriter, req Request, deps Deps) {
	clientID := strings.TrimSpace(ParamString(req.Params, "clientId"))
	if clientID == "" {
		RespondError(w, req.ID, "clientId is required")
		return
	}

	redirect, err := redirectURLForHost(deps.HTTPAddr, "localhost", "/")
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	brokerFlow, err := oauth.StartBrokerFlow(deps.Broker, redirect)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	creds := oauth.MicrosoftAppCredentials{
		ClientID:     clientID,
		ClientSecret: strings.TrimSpace(ParamString(req.Params, "clientSecret")),
		Tenant:       strings.TrimSpace(ParamString(req.Params, "tenant")),
	}

	flow, err := oauth.NewMicrosoftFlow(creds, brokerFlow)
	if err != nil {
		brokerFlow.Close()
		RespondError(w, req.ID, err.Error())
		return
	}

	deps.Flows.Register(flow)

	Respond(w, req.ID, map[string]any{
		"state":   flow.State(),
		"authUrl": flow.AuthURL(),
	})
}

func handleMicrosoftReauth(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	accountID := strings.TrimSpace(ParamString(req.Params, "accountId"))
	if accountID == "" {
		RespondError(w, req.ID, "accountId is required")
		return
	}

	creds, err := accounts.MicrosoftAppCreds(ctx, deps.Secrets, accountID)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	redirect, err := redirectURLForHost(deps.HTTPAddr, "localhost", "/")
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	brokerFlow, err := oauth.StartBrokerFlow(deps.Broker, redirect)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	flow, err := oauth.NewMicrosoftFlow(creds, brokerFlow)
	if err != nil {
		brokerFlow.Close()
		RespondError(w, req.ID, err.Error())
		return
	}

	deps.Flows.Register(flow)
	Respond(w, req.ID, map[string]any{
		"state":   flow.State(),
		"authUrl": flow.AuthURL(),
	})
}

func handleMicrosoftComplete(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	state := ParamString(req.Params, "state")
	if state == "" {
		RespondError(w, req.ID, "state is required")
		return
	}

	pending, ok := deps.Flows.Take(state)
	if !ok {
		RespondError(w, req.ID, "no pending microsoft flow for that state")
		return
	}
	flow, ok := pending.(*oauth.MicrosoftFlow)
	if !ok {
		RespondError(w, req.ID, "pending flow for that state is not a microsoft flow")
		return
	}

	waitCtx, cancel := context.WithTimeout(ctx, 6*time.Minute)
	defer cancel()

	tok, err := flow.Wait(waitCtx, 5*time.Minute)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	res, err := accounts.FinishMicrosoft(ctx, deps.Repo, deps.Secrets, flow, tok)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	publishAccountsChanged(deps, res.AccountID)
	Respond(w, req.ID, map[string]any{
		"accountId":   res.AccountID,
		"email":       res.DisplayName,
		"displayName": res.DisplayName,
	})
	kickAccountSync(deps, res.AccountID)
}

func handleCalDAVAdd(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	res, err := accounts.AddCalDAV(ctx, deps.Repo, deps.Secrets, accounts.CalDAVInput{
		URL:                ParamString(req.Params, "url"),
		Username:           ParamString(req.Params, "username"),
		Password:           ParamString(req.Params, "password"),
		DisplayName:        ParamString(req.Params, "displayName"),
		InsecureSkipVerify: ParamBool(req.Params, "insecureSkipVerify"),
	})
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	publishAccountsChanged(deps, res.AccountID)
	Respond(w, req.ID, map[string]any{"accountId": res.AccountID, "displayName": res.DisplayName})
	kickAccountSync(deps, res.AccountID)
}

func handleICalAdd(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	res, err := accounts.AddICal(ctx, deps.Repo, deps.Secrets, accounts.ICalInput{
		URL:         ParamString(req.Params, "url"),
		Username:    ParamString(req.Params, "username"),
		Password:    ParamString(req.Params, "password"),
		DisplayName: ParamString(req.Params, "displayName"),
	})
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	publishAccountsChanged(deps, res.AccountID)
	Respond(w, req.ID, map[string]any{"accountId": res.AccountID, "displayName": res.DisplayName})
	kickAccountSync(deps, res.AccountID)
}

func handleLocalAdd(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	res, err := accounts.AddLocal(ctx, deps.Repo, accounts.LocalInput{
		Root:        ParamString(req.Params, "root"),
		DisplayName: ParamString(req.Params, "displayName"),
	})
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	publishAccountsChanged(deps, res.AccountID)
	Respond(w, req.ID, map[string]any{"accountId": res.AccountID, "displayName": res.DisplayName})
	kickAccountSync(deps, res.AccountID)
}

func handleEvolutionAdd(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	res, err := accounts.AddEvolution(ctx, deps.Repo, accounts.EvolutionInput{
		DisplayName: ParamString(req.Params, "displayName"),
	})
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	publishAccountsChanged(deps, res.AccountID)
	Respond(w, req.ID, map[string]any{"accountId": res.AccountID, "displayName": res.DisplayName})
	kickAccountSync(deps, res.AccountID)
}

func kickAccountSync(deps Deps, accountID string) {
	if deps.Sync == nil {
		return
	}
	go func() {
		acc, err := deps.Repo.GetAccount(context.Background(), accountID)
		if err != nil {
			log.Warnf("post-add sync: %v", err)
			return
		}
		if err := deps.Sync.SyncAccount(context.Background(), acc); err != nil {
			log.Warnf("post-add sync %s: %v", accountID, err)
		}
	}()
}
