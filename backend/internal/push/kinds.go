// Package push sends Expo push notifications and manages device tokens.
//
// Layers: Client talks to the Expo Push API, Sender abstracts it (Fake for
// tests), Notifier resolves recipients from the database and cleans up invalid
// tokens, and the store functions (RegisterToken, DeleteToken) manage push_tokens.
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
)

// Kinds lists all known reminder kinds.
func Kinds() []string {
	return []string{
		KindLastPerson, KindWeatherChange, KindMedication, KindHelper,
		KindTrainingPlan, KindHealthDue, KindRehaCheckup, KindNewRequest,
		KindUrgentObservation,
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
