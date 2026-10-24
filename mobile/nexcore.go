// Package nexcore is the gomobile surface of the NexNews core for Android.
// Build: gomobile bind -target=android -androidapi 24 -javapkg=com.eirahoutmoss.nexnews.go -o android/app/libs/nexcore.aar ./mobile
// Every function exchanges strings (JSON), ints and bools only.
package nexcore

import "nexnews/core"

func Init(dataDir, lang string) error       { return core.MobileInit(dataDir, lang) }
func Refresh()                              { core.MobileRefresh() }
func SetCountry(code string)                { core.MobileSetCountry(code) }
func Weekly() string                        { return core.MobileWeekly() }
func Translate(max, seconds int)            { core.MobileTranslate(max, seconds) }
func LastFetch() int64                      { return core.MobileLastFetch() }
func IsFetching() bool                      { return core.MobileIsFetching() }
func Overview() string                      { return core.MobileOverview() }
func Category(name string) string           { return core.MobileCategory(name) }
func Story(link string) string              { return core.MobileStory(link) }
func Search(query string) string            { return core.MobileSearch(query) }
func Briefing() string                      { return core.MobileBriefing() }
func SourceHealth() string                  { return core.MobileSourceHealth() }
func ReadArticle(link string) string        { return core.MobileReadArticle(link) }
func Discover(site string) string           { return core.MobileDiscover(site) }
func Config() string                        { return core.MobileConfig() }
func SaveConfig(json string) error          { return core.MobileSaveConfig(json) }
func RestoreAllSources() int                { return core.MobileRestoreAllSources() }
func ToggleFav(itemJSON string) bool        { return core.MobileToggleFav(itemJSON) }
func FavLinks() string                      { return core.MobileFavLinks() }
func Favorites() string                     { return core.MobileFavorites() }
func History() string                       { return core.MobileHistory() }
func AddHistory(itemJSON string)            { core.MobileAddHistory(itemJSON) }
func ClearHistory()                         { core.MobileClearHistory() }
func ToggleFollowEvent(eventID string) bool { return core.MobileToggleFollowEvent(eventID) }
func NewAlerts() string                     { return core.MobileNewAlerts() }
func InQuietHours() bool                    { return core.MobileInQuietHours() }
func Radio() string                         { return core.MobileRadio() }
func LiveTV() string                        { return core.MobileLiveTV() }
func LiveHeaders(streamURL string) string   { return core.MobileLiveHeaders(streamURL) }
func Weather(place string) string           { return core.MobileWeather(place) }
func Finance() string                       { return core.MobileFinance() }
func Sports() string                        { return core.MobileSports() }
func Standings() string                     { return core.MobileStandings() }
func NetStats() string                      { return core.MobileNetStats() }
func AddTraffic(bytes int64)                { core.MobileAddTraffic(bytes) }
func Version() string                       { return core.MobileVersion() }

// RC85
func Followed() string                           { return core.MobileFollowed() }
func FollowedSeen()                              { core.MobileFollowedSeen() }
func Unfollow(link string)                       { core.MobileUnfollow(link) }
func ToggleFollowStory(link, id string) bool     { return core.MobileToggleFollowStory(link, id) }
func FollowCount() int                           { return core.MobileFollowCount() }
func SmartAlerts() string                        { return core.MobileSmartAlerts() }
func Usage() string                              { return core.MobileUsage() }
func SetDataMode(mode string, kbps, quotaMB int) { core.MobileSetDataMode(mode, kbps, quotaMB) }
func ResetUsage()                                { core.MobileResetUsage() }
func LiveState() string                          { return core.MobileLiveState() }
func SetLiveState(json string) error             { return core.MobileSetLiveState(json) }
func LiveHealth() string                         { return core.MobileLiveHealth() }
func ProbeLive(id, cands, tvg, files string) string {
	return core.MobileProbeLive(id, cands, tvg, files)
}
