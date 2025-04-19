package main

import (
	"fmt"
	"net/http"
	"strconv"

	"app/api"
	"app/csrf"
	"app/session"
	"app/view"
)

const cookieBackTo = "backTo"

func (h *handler) newEditPatch(w http.ResponseWriter, r *http.Request) error {
	rq := r.URL.Query()

	sid, err := strconv.ParseUint(rq.Get("subject_id"), 10, 64)
	if err != nil {
		http.Error(w, "subject_id must be a positive integer", http.StatusBadRequest)
		return nil
	}

	s := session.GetSession(r.Context())
	if s.UserID == 0 {
		http.SetCookie(w, &http.Cookie{
			Name:  cookieBackTo,
			Value: fmt.Sprintf("/suggest-subject?subject_id=%d", sid),
		})

		http.Redirect(w, r, "/login", http.StatusFound)
		return nil
	}

	var subject api.WikiSubject
	resp, err := h.client.R().SetResult(&subject).Get(fmt.Sprintf("https://next.bgm.tv/p1/wiki/subjects/%d", sid))
	if err != nil {
		return err
	}
	if resp.StatusCode() >= 300 {
		http.NotFound(w, r)
		return nil
	}

	return h.template.NewSubjectPatch.Execute(w, view.SubjectPatchEdit{
		PatchID:          "",
		SubjectID:        sid,
		CsrfToken:        csrf.GetToken(r),
		Reason:           "",
		Description:      "",
		Data:             subject,
		TurnstileSiteKey: h.config.TurnstileSiteKey,
	})
}
