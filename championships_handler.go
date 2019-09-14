package servermanager

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

type ChampionshipsHandler struct {
	*BaseHandler

	championshipManager *ChampionshipManager
}

func NewChampionshipsHandler(baseHandler *BaseHandler, championshipManager *ChampionshipManager) *ChampionshipsHandler {
	return &ChampionshipsHandler{
		BaseHandler:         baseHandler,
		championshipManager: championshipManager,
	}
}

// lists all available Championships known to Server Manager
func (ch *ChampionshipsHandler) list(w http.ResponseWriter, r *http.Request) {
	championships, err := ch.championshipManager.ListChampionships()

	if err != nil {
		logrus.Errorf("couldn't list championships, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ch.viewRenderer.MustLoadTemplate(w, r, "championships/index.html", map[string]interface{}{
		"championships": championships,
	})
}

// createOrEdit builds a Championship form for the user to create a Championship.
func (ch *ChampionshipsHandler) createOrEdit(w http.ResponseWriter, r *http.Request) {
	_, opts, err := ch.championshipManager.BuildChampionshipOpts(r)

	if err != nil {
		logrus.Errorf("couldn't build championship form, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ch.viewRenderer.MustLoadTemplate(w, r, "championships/new.html", opts)
}

// submit creates a given Championship and redirects the user to begin
// the flow of adding events to the new Championship
func (ch *ChampionshipsHandler) submit(w http.ResponseWriter, r *http.Request) {
	championship, edited, err := ch.championshipManager.HandleCreateChampionship(r)

	if err != nil {
		logrus.Errorf("couldn't create championship, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if edited {
		AddFlash(w, r, "Championship successfully edited!")
		http.Redirect(w, r, "/championship/"+championship.ID.String(), http.StatusFound)
	} else {
		AddFlash(w, r, "We've created the Championship. Now you need to add some Events!")
		http.Redirect(w, r, "/championship/"+championship.ID.String()+"/event", http.StatusFound)
	}
}

// view shows details of a given Championship
func (ch *ChampionshipsHandler) view(w http.ResponseWriter, r *http.Request) {
	championship, err := ch.championshipManager.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.Errorf("couldn't load championship, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	eventInProgress := false

	for _, event := range championship.Events {
		if event.InProgress() {
			eventInProgress = true
			break
		}
	}

	ch.viewRenderer.MustLoadTemplate(w, r, "championships/view.html", map[string]interface{}{
		"Championship":    championship,
		"EventInProgress": eventInProgress,
		"Account":         AccountFromRequest(r),
	})
}

// export returns all known data about a Championship in JSON format.
func (ch *ChampionshipsHandler) export(w http.ResponseWriter, r *http.Request) {
	championship, err := ch.championshipManager.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.Errorf("couldn't export championship, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// sign up responses are hidden for data protection reasons
	championship.SignUpForm.Responses = nil

	if !AccountFromRequest(r).HasGroupPrivilege(GroupWrite) {
		// if you don't have write access or above you can't see the replacement password
		championship.ReplacementPassword = ""
		championship.OverridePassword = false
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(championship)
}

// importChampionship reads Championship data from JSON.
func (ch *ChampionshipsHandler) importChampionship(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		championshipID, err := ch.championshipManager.ImportChampionship(r.FormValue("import"))

		if err != nil {
			logrus.Errorf("couldn't import championship, err: %s", err)
			AddErrorFlash(w, r, "Sorry, we couldn't import that championship! Check your JSON formatting.")
		} else {
			AddFlash(w, r, "Championship successfully imported!")
			http.Redirect(w, r, "/championship/"+championshipID, http.StatusFound)
		}
	}

	ch.viewRenderer.MustLoadTemplate(w, r, "championships/import-championship.html", nil)
}

type championshipResultsCollection struct {
	Name    string                `json:"name"`
	Results []championshipResults `json:"results"`
}

type championshipResults struct {
	Name string   `json:"name"`
	Log  []string `json:"log"`
}

// exportResults returns championship result files in JSON format.
func (ch *ChampionshipsHandler) exportResults(w http.ResponseWriter, r *http.Request) {
	championship, err := ch.championshipManager.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.Errorf("couldn't export championship, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var results []championshipResults

	for _, event := range championship.Events {

		if !event.Completed() {
			continue
		}

		var sessionFiles []string

		for _, session := range event.Sessions {
			sessionFiles = append(sessionFiles, session.Results.GetURL())
		}

		results = append(results, championshipResults{
			Name: "Event at " + prettifyName(event.RaceSetup.Track, false) + ", completed on " + event.CompletedTime.Format("Monday, January 2, 2006 3:04 PM (MST)"),
			Log:  sessionFiles,
		})
	}

	champResultsCollection := championshipResultsCollection{
		Name:    championship.Name,
		Results: results,
	}

	w.Header().Add("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(champResultsCollection)
}

// delete soft deletes a Championship.
func (ch *ChampionshipsHandler) delete(w http.ResponseWriter, r *http.Request) {
	err := ch.championshipManager.DeleteChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.Errorf("couldn't delete championship, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, "Championship deleted!")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ch *ChampionshipsHandler) eventImport(w http.ResponseWriter, r *http.Request) {
	championshipID := chi.URLParam(r, "championshipID")
	eventID := chi.URLParam(r, "eventID")

	if r.Method == http.MethodPost {
		err := ch.championshipManager.ImportEvent(championshipID, eventID, r)

		if err != nil {
			logrus.Errorf("Could not import championship event, error: %s", err)
			AddErrorFlash(w, r, "Could not import session files")
		} else {
			AddFlash(w, r, "Successfully imported session files!")
			http.Redirect(w, r, "/championship/"+championshipID, http.StatusFound)
			return
		}
	}

	event, results, err := ch.championshipManager.ListAvailableResultsFilesForEvent(championshipID, eventID)

	if err != nil {
		logrus.Errorf("Couldn't load session files, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ch.viewRenderer.MustLoadTemplate(w, r, "championships/import-event.html", map[string]interface{}{
		"Results":        results,
		"ChampionshipID": championshipID,
		"Event":          event,
	})
}

// eventConfiguration builds a Custom Race form with slight modifications
// to allow a user to configure a ChampionshipEvent.
func (ch *ChampionshipsHandler) eventConfiguration(w http.ResponseWriter, r *http.Request) {
	championshipRaceOpts, err := ch.championshipManager.BuildChampionshipEventOpts(r)

	if err != nil {
		logrus.Errorf("couldn't build championship race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ch.viewRenderer.MustLoadTemplate(w, r, "custom-race/new.html", championshipRaceOpts)
}

// submitEventConfiguration takes an Event Configuration from a form and
// builds an event optionally, this is used for editing ChampionshipEvents.
func (ch *ChampionshipsHandler) submitEventConfiguration(w http.ResponseWriter, r *http.Request) {
	championship, event, edited, err := ch.championshipManager.SaveChampionshipEvent(r)

	if err != nil {
		logrus.Errorf("couldn't build championship race, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if edited {
		AddFlash(w, r,
			fmt.Sprintf(
				"Championship race at %s was successfully edited!",
				prettifyName(event.RaceSetup.Track, false),
			),
		)
	} else {
		AddFlash(w, r,
			fmt.Sprintf(
				"Championship race at %s was successfully added!",
				prettifyName(event.RaceSetup.Track, false),
			),
		)
	}

	if r.FormValue("action") == "saveChampionship" {
		// end the race creation flow
		http.Redirect(w, r, "/championship/"+championship.ID.String(), http.StatusFound)
		return
	} else {
		// add another event
		http.Redirect(w, r, "/championship/"+championship.ID.String()+"/event", http.StatusFound)
	}
}

// startEvent begins a championship event given by its ID
func (ch *ChampionshipsHandler) startEvent(w http.ResponseWriter, r *http.Request) {
	err := ch.championshipManager.StartEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"), false)

	if err != nil {
		logrus.Errorf("Could not start championship event, err: %s", err)

		AddErrorFlash(w, r, "Couldn't start the Event")
	} else {
		AddFlash(w, r, "Event started successfully!")
		time.Sleep(time.Second * 1)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ch *ChampionshipsHandler) scheduleEvent(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		logrus.Errorf("couldn't parse schedule race form, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	championshipID := chi.URLParam(r, "championshipID")
	championshipEventID := chi.URLParam(r, "eventID")
	dateString := r.FormValue("event-schedule-date")
	timeString := r.FormValue("event-schedule-time")
	timezone := r.FormValue("event-schedule-timezone")

	location, err := time.LoadLocation(timezone)

	if err != nil {
		logrus.WithError(err).Errorf("could not find location: %s", location)
		location = time.Local
	}

	// Parse time in correct time zone
	date, err := time.ParseInLocation("2006-01-02-15:04", dateString+"-"+timeString, location)

	if err != nil {
		logrus.Errorf("couldn't parse schedule championship event date, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = ch.championshipManager.ScheduleEvent(championshipID, championshipEventID, date, r.FormValue("action"))

	if err != nil {
		logrus.Errorf("couldn't schedule championship event, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, fmt.Sprintf("We have scheduled the Championship Event to begin at %s", date.Format(time.RFC1123)))
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ch *ChampionshipsHandler) scheduleEventRemove(w http.ResponseWriter, r *http.Request) {
	err := ch.championshipManager.ScheduleEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"),
		time.Time{}, "remove")

	if err != nil {
		logrus.Errorf("couldn't schedule championship event, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

// deleteEvent soft deletes a championship event
func (ch *ChampionshipsHandler) deleteEvent(w http.ResponseWriter, r *http.Request) {
	err := ch.championshipManager.DeleteEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"))

	if err != nil {
		logrus.Errorf("Could not delete championship event, err: %s", err)

		AddErrorFlash(w, r, "Couldn't delete the Event")
	} else {
		AddFlash(w, r, "Event deleted successfully!")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

// startPracticeEvent starts a Practice session for a given event
func (ch *ChampionshipsHandler) startPracticeEvent(w http.ResponseWriter, r *http.Request) {
	err := ch.championshipManager.StartPracticeEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"))

	if err != nil {
		logrus.Errorf("Could not start practice championship event, err: %s", err)

		AddErrorFlash(w, r, "Couldn't start the Practice Event")
	} else {
		AddFlash(w, r, "Practice Event started successfully!")
		time.Sleep(time.Second * 1)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

// cancelEvent stops a running championship event and clears any saved results
func (ch *ChampionshipsHandler) cancelEvent(w http.ResponseWriter, r *http.Request) {
	err := ch.championshipManager.CancelEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"))

	if err != nil {
		logrus.Errorf("Could not cancel championship event, err: %s", err)

		AddErrorFlash(w, r, "Couldn't cancel the Championship Event")
	} else {
		AddFlash(w, r, "Event cancelled successfully!")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

// restartEvent stops a running championship event and clears any saved results
// then starts the event again.
func (ch *ChampionshipsHandler) restartEvent(w http.ResponseWriter, r *http.Request) {
	err := ch.championshipManager.RestartEvent(chi.URLParam(r, "championshipID"), chi.URLParam(r, "eventID"))

	if err != nil {
		logrus.Errorf("Could not restart championship event, err: %s", err)

		AddErrorFlash(w, r, "Couldn't restart the Championship Event")
	} else {
		AddFlash(w, r, "Event restarted successfully!")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ch *ChampionshipsHandler) icalFeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Add("Content-Disposition", "inline; filename=championship.ics")

	err := ch.championshipManager.BuildICalFeed(chi.URLParam(r, "championshipID"), w)

	if err != nil {
		logrus.WithError(err).Error("could not build scheduled races feed")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (ch *ChampionshipsHandler) driverPenalty(w http.ResponseWriter, r *http.Request) {
	err := ch.championshipManager.ModifyDriverPenalty(
		chi.URLParam(r, "championshipID"),
		chi.URLParam(r, "classID"),
		chi.URLParam(r, "driverGUID"),
		PenaltyAction(r.FormValue("action")),
		formValueAsInt(r.FormValue("PointsPenalty")),
	)

	if err != nil {
		logrus.Errorf("Could not modify championship driver penalty, err: %s", err)

		AddErrorFlash(w, r, "Couldn't modify driver penalty")
	} else {
		AddFlash(w, r, "Driver penalty successfully modified")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ch *ChampionshipsHandler) teamPenalty(w http.ResponseWriter, r *http.Request) {
	err := ch.championshipManager.ModifyTeamPenalty(
		chi.URLParam(r, "championshipID"),
		chi.URLParam(r, "classID"),
		chi.URLParam(r, "team"),
		PenaltyAction(r.FormValue("action")),
		formValueAsInt(r.FormValue("PointsPenalty")),
	)

	if err != nil {
		logrus.Errorf("Could not modify championship penalty, err: %s", err)

		AddErrorFlash(w, r, "Couldn't modify team penalty")
	} else {
		AddFlash(w, r, "Team penalty successfully modified")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

type entrantSlot struct {
	Size     int
	Capacity int
}

func (ch *ChampionshipsHandler) signUpForm(w http.ResponseWriter, r *http.Request) {
	championship, opts, err := ch.championshipManager.BuildChampionshipOpts(r)

	if err != nil {
		logrus.WithError(err).Error("couldn't load championship")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !championship.SignUpForm.Enabled {
		http.NotFound(w, r)
		return
	}

	signedUpEntrants := make(map[string]*entrantSlot)

	for _, entrant := range championship.AllEntrants() {
		if _, ok := signedUpEntrants[entrant.Model]; !ok {
			signedUpEntrants[entrant.Model] = &entrantSlot{}
		}

		signedUpEntrants[entrant.Model].Capacity++

		if entrant.GUID != "" {
			signedUpEntrants[entrant.Model].Size++
		}
	}

	account := AccountFromRequest(r)

	if account != OpenAccount {
		opts["FormData"] = &ChampionshipSignUpResponse{
			Name: account.DriverName,
			GUID: account.GUID,
			Team: account.Team,
		}
	} else {
		opts["FormData"] = &ChampionshipSignUpResponse{}
	}

	opts["SignedUpEntrants"] = signedUpEntrants

	if r.Method == http.MethodPost {
		signUpResponse, foundSlot, err := ch.championshipManager.HandleChampionshipSignUp(r)

		if err != nil {
			switch err.(type) {
			case ValidationError:
				opts["FormData"] = signUpResponse
				opts["ValidationError"] = err.Error()
			default:
				logrus.WithError(err).Error("couldn't handle championship")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		} else {
			if championship.SignUpForm.RequiresApproval {
				AddFlash(w, r, "Thanks for registering for the championship! Your registration is pending approval by an administrator.")
				http.Redirect(w, r, "/championship/"+championship.ID.String(), http.StatusFound)
				return
			} else {
				if foundSlot {
					AddFlash(w, r, "Thanks for registering for the championship!")
					http.Redirect(w, r, "/championship/"+championship.ID.String(), http.StatusFound)
					return
				} else {
					opts["FormData"] = signUpResponse
					opts["ValidationError"] = fmt.Sprintf("There are no more available slots for the car: %s. Please pick a different car.", prettifyName(signUpResponse.GetCar(), true))
				}
			}
		}
	}

	ch.viewRenderer.MustLoadTemplate(w, r, "championships/sign-up.html", opts)
}

func (ch *ChampionshipsHandler) signedUpEntrants(w http.ResponseWriter, r *http.Request) {
	championship, err := ch.championshipManager.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.WithError(err).Error("couldn't load championship")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !championship.SignUpForm.Enabled {
		http.NotFound(w, r)
		return
	}

	sort.Slice(championship.SignUpForm.Responses, func(i, j int) bool {
		return championship.SignUpForm.Responses[i].Created.After(championship.SignUpForm.Responses[j].Created)
	})

	ch.viewRenderer.MustLoadTemplate(w, r, "championships/signed-up-entrants.html", map[string]interface{}{
		"Championship": championship,
	})
}

func (ch *ChampionshipsHandler) signedUpEntrantsCSV(w http.ResponseWriter, r *http.Request) {
	championship, err := ch.championshipManager.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.WithError(err).Error("couldn't load championship")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	headers := []string{
		"Created",
		"Name",
		"Team",
		"GUID",
		"Email",
		"Car",
		"Skin",
		"Status",
	}

	for _, question := range championship.SignUpForm.ExtraFields {
		headers = append(headers, question)
	}

	var out [][]string

	out = append(out, headers)

	for _, entrant := range championship.SignUpForm.Responses {
		data := []string{
			entrant.Created.String(),
			entrant.Name,
			entrant.Team,
			entrant.GUID,
			entrant.Email,
			entrant.Car,
			entrant.Skin,
			string(entrant.Status),
		}

		for _, question := range championship.SignUpForm.ExtraFields {
			if response, ok := entrant.Questions[question]; ok {
				data = append(data, response)
			} else {
				data = append(data, "")
			}
		}

		out = append(out, data)
	}

	w.Header().Add("Content-Type", "text/csv")
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment;filename=Entrants_%s.csv", championship.Name))
	wr := csv.NewWriter(w)
	wr.UseCRLF = true
	_ = wr.WriteAll(out)
}

func (ch *ChampionshipsHandler) modifyEntrantStatus(w http.ResponseWriter, r *http.Request) {
	championship, err := ch.championshipManager.LoadChampionship(chi.URLParam(r, "championshipID"))

	if err != nil {
		logrus.WithError(err).Error("couldn't load championship")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !championship.SignUpForm.Enabled {
		http.NotFound(w, r)
		return
	}

	entrantGUID := chi.URLParam(r, "entrantGUID")

	for index, entrant := range championship.SignUpForm.Responses {
		if entrant.GUID != entrantGUID {
			continue
		}

		switch r.URL.Query().Get("action") {
		case "accept":
			if entrant.Status == ChampionshipEntrantAccepted {
				AddFlash(w, r, "This entrant has already been accepted.")
				break
			}

			// add the entrant to the entrylist
			foundSlot, _, err := ch.championshipManager.AddEntrantFromSessionData(championship, entrant, true, championship.SignUpForm.HideCarChoice)

			if err != nil {
				logrus.WithError(err).Error("couldn't add entrant to championship")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			if foundSlot {
				entrant.Status = ChampionshipEntrantAccepted

				AddFlash(w, r, "The entrant was successfully accepted!")
			} else {
				AddErrorFlash(w, r, "There are no more slots available for the given entrant and car. Please check the Championship configuration.")
			}
		case "reject":
			entrant.Status = ChampionshipEntrantRejected
			championship.ClearEntrant(entrantGUID)
		case "delete":
			championship.SignUpForm.Responses = append(championship.SignUpForm.Responses[:index], championship.SignUpForm.Responses[index+1:]...)
			championship.ClearEntrant(entrantGUID)

		default:
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	if err := ch.championshipManager.UpsertChampionship(championship); err != nil {
		logrus.WithError(err).Error("couldn't save championship")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
