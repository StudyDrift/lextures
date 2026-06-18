package gamification

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed xp_awards.json
var xpAwardsJSON []byte

var (
	awardsOnce sync.Once
	awardsMap  map[string]int
	awardsErr  error
)

// ActivityType constants for XP awards.
const (
	ActivityModuleItemViewed = "module_item_viewed"
	ActivityQuizPassed       = "quiz_passed"
	ActivityCourseCompleted  = "course_completed"
	ActivityPathCompleted    = "path_completed"
)

// BadgeType constants for milestone badges.
const (
	BadgeStreak7            = "streak_7"
	BadgeStreak30           = "streak_30"
	BadgeXP100              = "xp_100"
	BadgeXP1000             = "xp_1000"
	BadgeFirstCourseComplete = "first_course_complete"
)

// StreakFreezeMilestones award one freeze when the streak first reaches each value.
var StreakFreezeMilestones = []int{7, 30, 60}

func loadAwards() {
	awardsOnce.Do(func() {
		awardsMap = make(map[string]int)
		awardsErr = json.Unmarshal(xpAwardsJSON, &awardsMap)
	})
}

// XPAward returns configured XP for an activity type, or 0 when unknown.
func XPAward(activityType string) int {
	loadAwards()
	if awardsErr != nil {
		return 0
	}
	return awardsMap[activityType]
}

// LevelFromXP computes level = floor(sqrt(xp / 10)).
func LevelFromXP(xp int) int {
	if xp <= 0 {
		return 0
	}
	lvl := 0
	for (lvl+1)*(lvl+1)*10 <= xp {
		lvl++
	}
	return lvl
}

// XPForNextLevel returns the XP threshold for the next level.
func XPForNextLevel(level int) int {
	return (level + 1) * (level + 1) * 10
}
