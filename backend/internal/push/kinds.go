// Package push sends push notifications to native devices and browsers and manages
// their tokens.
//
// Layers: Client talks to APNs (APNSClient) and FCM (FCMClient) directly, Sender
// abstracts it (Fake for tests), WebClient does the same for Web Push, Notifier
// resolves recipients from the database and cleans up invalid tokens, and the store
// functions (RegisterToken, DeleteToken) manage push_tokens.
package push

// Reminder kinds. They are the values of reminder_settings.kind and the kind
// argument of Notifier.NotifyUsers.
const (
	KindLastPerson        = "last_person"
	KindWeatherChange     = "weather_change"
	KindMedication        = "medication"
	KindHelper            = "helper"
	KindTrainingPlan      = "training_plan"
	KindHealthDue         = "health_due"
	KindRehaCheckup       = "reha_checkup"
	KindNewRequest        = "new_request"
	KindUrgentObservation = "urgent_observation"
	// KindObservation is a new observation ("Auffälligkeit") that is not urgent: info or "please check".
	KindObservation = "observation"
)

// Kinds lists all known reminder kinds.
func Kinds() []string {
	return []string{
		KindLastPerson, KindWeatherChange, KindMedication, KindHelper,
		KindTrainingPlan, KindHealthDue, KindRehaCheckup, KindNewRequest,
		KindUrgentObservation, KindObservation,
	}
}

// ValidKind reports whether kind is a known reminder kind.
func ValidKind(kind string) bool {
	for _, k := range Kinds() {
		if k == kind {
			return true
		}
	}
	return false
}

// OptIn reports whether users must enable the kind themselves. Every other kind
// is on by default and can be switched off; an opt-in kind is off until the user
// has a reminder_settings row with enabled = true. Currently only KindNewRequest
// (a push to the whole stable for every new request would be noise by default).
func OptIn(kind string) bool { return kind == KindNewRequest }
