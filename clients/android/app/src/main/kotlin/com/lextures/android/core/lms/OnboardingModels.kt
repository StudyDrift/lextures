package com.lextures.android.core.lms

import com.lextures.android.R
import kotlinx.serialization.Serializable

enum class PriorKnowledgeLevel(val wire: String) {
    Beginner("beginner"),
    Intermediate("intermediate"),
    Advanced("advanced"),
}

@Serializable
data class OnboardingStatus(
    val completed: Boolean = false,
    val step: Int = 0,
    val shouldShowFlow: Boolean = true,
)

@Serializable
data class LearnerGoals(
    val id: String = "",
    val userId: String = "",
    val topic: String = "",
    val goalText: String? = null,
    val targetDate: String? = null,
    val dailyMinutes: Int = 20,
    val priorKnowledgeLevel: String = PriorKnowledgeLevel.Beginner.wire,
    val diagnosticScore: Double? = null,
    val diagnosticSkipped: Boolean = false,
    val onboardingStep: Int = 0,
    val onboardingCompleted: Boolean = false,
    val reminderOptIn: Boolean = false,
    val reminderTime: String? = null,
    val recommendedCourseCode: String? = null,
    val recommendedCourseTitle: String? = null,
)

@Serializable
data class DiagnosticQuestion(
    val id: String,
    val prompt: String,
    val choices: List<String> = emptyList(),
)

@Serializable
data class GoalsEnvelope(
    val goals: LearnerGoals,
)

@Serializable
data class DiagnosticQuestionsResponse(
    val questions: List<DiagnosticQuestion> = emptyList(),
)

data class OnboardingTopic(
    val id: String,
    val labelRes: Int,
)

val onboardingTopics: List<OnboardingTopic> = listOf(
    OnboardingTopic("python", R.string.mobile_onboarding_topic_python),
    OnboardingTopic("javascript", R.string.mobile_onboarding_topic_javascript),
    OnboardingTopic("data-science", R.string.mobile_onboarding_topic_dataScience),
    OnboardingTopic("math", R.string.mobile_onboarding_topic_math),
    OnboardingTopic("business", R.string.mobile_onboarding_topic_business),
    OnboardingTopic("design", R.string.mobile_onboarding_topic_design),
)

enum class OnboardingStep(val value: Int) {
    Welcome(0),
    Topic(1),
    Experience(2),
    Diagnostic(3),
    Habits(4),
    Consent(5),
    Complete(6),
}
