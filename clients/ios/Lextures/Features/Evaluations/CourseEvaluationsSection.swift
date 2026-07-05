import SwiftUI

/// Course evaluations workspace section (M7.7).
struct CourseEvaluationsSection: View {
    let course: CourseSummary
    var showResults: Bool = false

    var body: some View {
        if course.viewerIsStaff || showResults {
            EvaluationResultsView(course: course)
        } else {
            EvaluationFormView(course: course)
        }
    }
}
