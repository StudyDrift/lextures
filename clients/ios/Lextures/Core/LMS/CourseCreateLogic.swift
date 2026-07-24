import Foundation

/// Course create wizard helpers (M11.5 / MOB.1) — permission gate, templates, validation, step state.
enum CourseCreateLogic {
    static let courseCreatePermission = "global:app:course:create"
    static let blankTemplateId = "blank"
    static let defaultFirstModuleTitle = "Getting started"
    static let defaultTemplateId = "higher-ed-15-week"

    enum CourseMode: String, CaseIterable, Identifiable, Hashable {
        case traditional
        case competencyBased = "competency_based"

        var id: String { rawValue }

        var labelKey: String {
            switch self {
            case .traditional: return "mobile.createCourse.mode.traditional"
            case .competencyBased: return "mobile.createCourse.mode.competency"
            }
        }
    }

    enum CreateSource: String, CaseIterable, Identifiable, Hashable {
        case scratch
        case canvas

        var id: String { rawValue }

        var titleKey: String {
            switch self {
            case .scratch: return "mobile.createCourse.source.scratch.title"
            case .canvas: return "mobile.createCourse.source.canvas.title"
            }
        }

        var summaryKey: String {
            switch self {
            case .scratch: return "mobile.createCourse.source.scratch.summary"
            case .canvas: return "mobile.createCourse.source.canvas.summary"
            }
        }
    }

    enum AssessmentKind: String, CaseIterable, Identifiable, Hashable, Codable {
        case quiz
        case assignment

        var id: String { rawValue }

        var labelKey: String {
            switch self {
            case .quiz: return "mobile.createCourse.competency.assessment.quiz"
            case .assignment: return "mobile.createCourse.competency.assessment.assignment"
            }
        }
    }

    struct SubOutcomeDraft: Identifiable, Equatable, Hashable, Codable {
        var id: String
        var title: String
        var description: String
        var assessmentTitle: String
        var assessmentKind: AssessmentKind

        init(
            id: String = UUID().uuidString.lowercased(),
            title: String = "",
            description: String = "",
            assessmentTitle: String = "",
            assessmentKind: AssessmentKind = .quiz
        ) {
            self.id = id
            self.title = title
            self.description = description
            self.assessmentTitle = assessmentTitle
            self.assessmentKind = assessmentKind
        }

        static func empty() -> SubOutcomeDraft { SubOutcomeDraft() }
    }

    struct CompetencyDraft: Identifiable, Equatable, Hashable, Codable {
        var id: String
        var title: String
        var description: String
        var subOutcomes: [SubOutcomeDraft]
        var expanded: Bool

        init(
            id: String = UUID().uuidString.lowercased(),
            title: String = "",
            description: String = "",
            subOutcomes: [SubOutcomeDraft] = [SubOutcomeDraft.empty()],
            expanded: Bool = true
        ) {
            self.id = id
            self.title = title
            self.description = description
            self.subOutcomes = subOutcomes
            self.expanded = expanded
        }

        static func empty() -> CompetencyDraft { CompetencyDraft() }
    }

    enum WizardStep: Int, CaseIterable, Identifiable, Comparable {
        case source = 0
        case basics = 1
        case syllabus = 2
        case finish = 3
        case features = 4

        var id: Int { rawValue }

        /// Total progress steps shown in the header (excludes source chooser).
        static let totalProgressSteps = 4

        static func < (lhs: WizardStep, rhs: WizardStep) -> Bool {
            lhs.rawValue < rhs.rawValue
        }

        /// Progress steps shown in the header (excludes source chooser).
        static var progressSteps: [WizardStep] { [.basics, .syllabus, .finish, .features] }

        var labelKey: String {
            switch self {
            case .source: return "mobile.createCourse.step.source"
            case .basics: return "mobile.createCourse.step.basics"
            case .syllabus: return "mobile.createCourse.step.syllabus"
            case .finish: return "mobile.createCourse.step.module"
            case .features: return "mobile.createCourse.step.features"
            }
        }

        func finishLabelKey(isCompetency: Bool) -> String {
            if self == .finish && isCompetency {
                return "mobile.createCourse.step.competencies"
            }
            return labelKey
        }
    }

    struct TemplateSection: Equatable, Hashable {
        var heading: String
        var markdown: String
    }

    struct StarterTemplate: Identifiable, Equatable, Hashable {
        var id: String
        var nameKey: String
        var summaryKey: String
        var suggestedFirstModuleTitle: String
        var sections: [TemplateSection]
    }

    /// Port of web `COURSE_CREATE_STARTER_TEMPLATES` (ids + content parity).
    static let starterTemplates: [StarterTemplate] = [
        StarterTemplate(
            id: "k12-semester",
            nameKey: "mobile.createCourse.template.k12.name",
            summaryKey: "mobile.createCourse.template.k12.summary",
            suggestedFirstModuleTitle: "Unit 1: Getting started",
            sections: [
                TemplateSection(
                    heading: "Course overview",
                    markdown: "Briefly describe what students will learn this term and how day-to-day class time is structured.\n\n- **Big ideas**:\n- **Major projects or exams**:\n"
                ),
                TemplateSection(
                    heading: "Materials & technology",
                    markdown: "List required texts, supplies, and any accounts or apps (including this LMS).\n\n| Item | Notes |\n|------|-------|\n| | |\n"
                ),
                TemplateSection(
                    heading: "Grading",
                    markdown: "Explain how the gradebook categories work and how families can check progress.\n\n- **Formative vs summative**:\n- **Late work**:\n- **Retakes or revisions**:\n"
                ),
                TemplateSection(
                    heading: "Classroom expectations",
                    markdown: "Norms for participation, discussion, academic honesty, and communication.\n\n" +
                        "1. **Respect** — listen and assume good intent.\n2. **Readiness** — arrive with materials.\n3. **Integrity** — cite sources; complete your own work.\n"
                ),
                TemplateSection(
                    heading: "Contact & support",
                    markdown: "Best way to reach you, typical response time, and how to request extra help or accommodations.\n\n- **Email**:\n- **Office hours**:\n- **School resources**:\n"
                ),
            ]
        ),
        StarterTemplate(
            id: "higher-ed-15-week",
            nameKey: "mobile.createCourse.template.higherEd.name",
            summaryKey: "mobile.createCourse.template.higherEd.summary",
            suggestedFirstModuleTitle: "Week 1: Introduction & syllabus",
            sections: [
                TemplateSection(
                    heading: "Instructor & meeting times",
                    markdown: "**Instructor:**\n\n**Email:**\n\n**Office hours:**\n\n**Lecture / lab / discussion:**\n\n**Course site:** This LMS\n"
                ),
                TemplateSection(
                    heading: "Course description",
                    markdown: "Paste the catalog description, then add a short paragraph on themes and prerequisites.\n\n**Prerequisites:**\n\n**Credit hours:**\n"
                ),
                TemplateSection(
                    heading: "Learning outcomes",
                    markdown: "By the end of the term, students will be able to:\n\n1. \n2. \n3. \n"
                ),
                TemplateSection(
                    heading: "Schedule at a glance",
                    markdown: "Outline major units or themes by week. Adjust dates to match your term.\n\n| Week | Topics | Due |\n|------|--------|-----|\n| 1 | | |\n| 2 | | |\n| … | | |\n"
                ),
                TemplateSection(
                    heading: "Assessment & grading",
                    markdown: "Summarize weights (they should match your gradebook assignment groups).\n\n- **Exams:**\n- **Assignments:**\n- **Participation:**\n\n**Curve / grading scale:**\n"
                ),
                TemplateSection(
                    heading: "Policies",
                    markdown: "Attendance, late work, academic integrity, accessibility, and technology expectations. Link or summarize institutional policies as required.\n"
                ),
            ]
        ),
        StarterTemplate(
            id: "self-paced",
            nameKey: "mobile.createCourse.template.selfPaced.name",
            summaryKey: "mobile.createCourse.template.selfPaced.summary",
            suggestedFirstModuleTitle: "Start here",
            sections: [
                TemplateSection(
                    heading: "How this course works",
                    markdown: "Explain that learners move at their own speed and where modules, due dates (if any), and checkpoints live.\n\n" +
                        "- **Estimated time:**\n- **Recommended pace:**\n- **Hard deadlines (if any):**\n"
                ),
                TemplateSection(
                    heading: "Learning goals",
                    markdown: "What someone should be able to do after finishing all modules.\n\n1. \n2. \n3. \n"
                ),
                TemplateSection(
                    heading: "Getting help",
                    markdown: "Where to ask questions (feed, inbox, discussion), expected response times, and links to FAQs or community norms.\n"
                ),
                TemplateSection(
                    heading: "Completion criteria",
                    markdown: "Define what “done” means: required items, minimum scores, portfolio review, or certificate triggers.\n"
                ),
            ]
        ),
        StarterTemplate(
            id: "bootcamp",
            nameKey: "mobile.createCourse.template.bootcamp.name",
            summaryKey: "mobile.createCourse.template.bootcamp.summary",
            suggestedFirstModuleTitle: "Day 1: Orientation",
            sections: [
                TemplateSection(
                    heading: "Program overview",
                    markdown: "Length of program, daily schedule blocks, and how cohorts or teams are organized.\n\n- **Start / end dates:**\n- **Live vs async:**\n- **Capstone or demo day:**\n"
                ),
                TemplateSection(
                    heading: "Projects & deliverables",
                    markdown: "List major builds, presentations, or assessments with rough timing.\n\n| Milestone | Description | Target |\n|-----------|-------------|--------|\n| | | |\n"
                ),
                TemplateSection(
                    heading: "Tools & environment",
                    markdown: "Required installs, repos, API keys (never commit secrets), and how to verify your setup.\n\n```bash\n# Example: clone and install\n```\n"
                ),
                TemplateSection(
                    heading: "Code of conduct & academic integrity",
                    markdown: "Collaboration rules, AI use policy if applicable, harassment-free space, and escalation paths.\n"
                ),
            ]
        ),
        StarterTemplate(
            id: "onboarding",
            nameKey: "mobile.createCourse.template.onboarding.name",
            summaryKey: "mobile.createCourse.template.onboarding.summary",
            suggestedFirstModuleTitle: "Week one checklist",
            sections: [
                TemplateSection(
                    heading: "Welcome",
                    markdown: "Warm intro to the program, who to ping first, and what success looks like in the first 30 days.\n"
                ),
                TemplateSection(
                    heading: "Role & expectations",
                    markdown: "Responsibilities, stakeholders, working agreements, and how performance is reviewed.\n"
                ),
                TemplateSection(
                    heading: "Systems & access",
                    markdown: "Accounts, security basics, where docs live, and how to request access.\n\n- [ ] Email / SSO\n- [ ] Chat\n- [ ] Ticketing\n"
                ),
                TemplateSection(
                    heading: "First week checklist",
                    markdown: "Concrete tasks with owners and links.\n\n1. Complete profile in this LMS\n2. Read handbook section …\n3. Schedule intro 1:1s\n"
                ),
            ]
        ),
    ]

    static let gradeLevels: [String] = CourseSettingsLogic.gradeLevels

    static func createCourseEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileCreateCourse || features.ffMobileCourseCreateV2
    }

    static func courseCreateV2Enabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileCourseCreateV2
    }

    static func canvasImportEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileCanvasImport
    }

    static func shouldShowCanvasImportSource(
        permissions: [String],
        features: MobilePlatformFeatures,
        isOnline: Bool
    ) -> Bool {
        CanvasImportLogic.shouldShowCanvasImportEntry(
            permissions: permissions,
            features: features,
            isOnline: isOnline
        )
    }

    static func canCreateCourses(permissions: [String]) -> Bool {
        permissions.contains(courseCreatePermission)
    }

    static func shouldShowNewCourseAction(
        permissions: [String],
        features: MobilePlatformFeatures,
        isOnline: Bool
    ) -> Bool {
        guard createCourseEnabled(features) else { return false }
        guard canCreateCourses(permissions: permissions) else { return false }
        return isOnline
    }

    static func initialWizardStep(v2Enabled: Bool) -> WizardStep {
        v2Enabled ? .source : .basics
    }

    static func validateTitle(_ title: String) -> String? {
        if title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return "mobile.createCourse.error.titleRequired"
        }
        return nil
    }

    /// Parity with web `validateCompetencies` (localized message keys + format args).
    struct CompetencyValidationError: Equatable {
        var key: String
        var args: [String]

        static func message(_ key: String, _ args: String...) -> CompetencyValidationError {
            CompetencyValidationError(key: key, args: args)
        }
    }

    static func validateCompetencies(_ competencies: [CompetencyDraft]) -> CompetencyValidationError? {
        if competencies.isEmpty {
            return .message("mobile.createCourse.error.competency.minOne")
        }
        for (compIndex, competency) in competencies.enumerated() {
            let title = competency.title.trimmingCharacters(in: .whitespacesAndNewlines)
            if title.isEmpty {
                return .message("mobile.createCourse.error.competency.titleRequired", "\(compIndex + 1)")
            }
            if competency.subOutcomes.isEmpty {
                return .message("mobile.createCourse.error.competency.subOutcomeMinOne", title)
            }
            for (subIndex, subOutcome) in competency.subOutcomes.enumerated() {
                let subTitle = subOutcome.title.trimmingCharacters(in: .whitespacesAndNewlines)
                if subTitle.isEmpty {
                    return .message(
                        "mobile.createCourse.error.competency.subOutcomeTitleRequired",
                        title,
                        "\(subIndex + 1)"
                    )
                }
                if subOutcome.assessmentTitle.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    return .message("mobile.createCourse.error.competency.assessmentTitleRequired", subTitle)
                }
            }
        }
        return nil
    }

    static func template(for id: String) -> StarterTemplate? {
        starterTemplates.first { $0.id == id }
    }

    static func suggestedFirstModuleTitle(templateId: String, existing: String) -> String {
        let trimmed = existing.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty { return trimmed }
        if templateId == blankTemplateId { return defaultFirstModuleTitle }
        return template(for: templateId)?.suggestedFirstModuleTitle ?? defaultFirstModuleTitle
    }

    /// Assigns ids when applying a template (parity with web `templateSectionsToSyllabus`).
    static func templateSectionsToSyllabus(_ sections: [TemplateSection]) -> [SyllabusSection] {
        sections.map { section in
            SyllabusSection(
                id: UUID().uuidString.lowercased(),
                heading: section.heading,
                markdown: section.markdown
            )
        }
    }

    static func shouldPatchSyllabus(templateId: String) -> Bool {
        templateId != blankTemplateId && template(for: templateId) != nil
    }

    /// True when Step 1 should update an existing course rather than POST a new one.
    static func shouldUpdateExistingCourse(createdCourseCode: String?) -> Bool {
        guard let code = createdCourseCode?.trimmingCharacters(in: .whitespacesAndNewlines) else {
            return false
        }
        return !code.isEmpty
    }

    static func shouldConfirmCancel(createdCourseCode: String?) -> Bool {
        shouldUpdateExistingCourse(createdCourseCode: createdCourseCode)
    }

    static func buildCreateRequest(
        title: String,
        description: String,
        mode: CourseMode,
        termId: String?,
        gradeLevel: String?
    ) -> CreateCourseRequest {
        let term = termId?.trimmingCharacters(in: .whitespacesAndNewlines)
        let grade = gradeLevel?.trimmingCharacters(in: .whitespacesAndNewlines)
        return CreateCourseRequest(
            title: title.trimmingCharacters(in: .whitespacesAndNewlines),
            description: description.trimmingCharacters(in: .whitespacesAndNewlines),
            courseType: mode.rawValue,
            termId: (term?.isEmpty == false) ? term : nil,
            gradeLevel: (grade?.isEmpty == false) ? grade : nil
        )
    }

    /// PUT body for re-entering Basics after the course exists (parity with web `putBodyFromCourse`).
    static func buildUpdateRequest(
        course: CourseSummary,
        title: String,
        description: String,
        termId: String?,
        gradeLevel: String?
    ) -> CourseUpdateRequest {
        let mode = course.scheduleMode == "relative" ? "relative" : "fixed"
        let term = termId?.trimmingCharacters(in: .whitespacesAndNewlines)
        let grade = gradeLevel?.trimmingCharacters(in: .whitespacesAndNewlines)
        return CourseUpdateRequest(
            title: title.trimmingCharacters(in: .whitespacesAndNewlines),
            description: description.trimmingCharacters(in: .whitespacesAndNewlines),
            published: course.published ?? false,
            startsAt: course.startsAt,
            endsAt: course.endsAt,
            visibleFrom: course.visibleFrom,
            hiddenAt: course.hiddenAt,
            scheduleMode: mode,
            relativeEndAfter: course.relativeEndAfter,
            relativeHiddenAfter: course.relativeHiddenAfter,
            courseHomeLanding: course.courseHomeLanding ?? "data",
            courseHomeContentItemId: course.courseHomeContentItemId,
            courseTimezone: course.courseTimezone,
            gradeLevel: (grade?.isEmpty == false) ? grade : nil,
            termId: (term?.isEmpty == false) ? term : nil
        )
    }

    static func modeFromCourseType(_ courseType: String?) -> CourseMode {
        if courseType == CourseMode.competencyBased.rawValue {
            return .competencyBased
        }
        return .traditional
    }

    /// Decode `org_id` from a JWT payload without signature verification.
    static func orgIdFromAccessToken(_ jwt: String) -> String? {
        let segments = jwt.split(separator: ".")
        guard segments.count >= 2 else { return nil }
        var base64 = String(segments[1])
            .replacingOccurrences(of: "-", with: "+")
            .replacingOccurrences(of: "_", with: "/")
        while base64.count % 4 != 0 { base64.append("=") }
        guard
            let data = Data(base64Encoded: base64),
            let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
        else { return nil }
        if let org = json["org_id"] as? String, !org.isEmpty { return org }
        if let org = json["orgId"] as? String, !org.isEmpty { return org }
        return nil
    }

    static func resolveOrgId(accessToken: String?, courses: [CourseSummary]) -> String? {
        if let token = accessToken, let fromJwt = orgIdFromAccessToken(token) {
            return fromJwt
        }
        return courses.compactMap(\.orgId).first { !$0.isEmpty }
    }
}
