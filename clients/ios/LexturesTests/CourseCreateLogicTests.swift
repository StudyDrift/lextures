import XCTest
@testable import Lextures

final class CourseCreateLogicTests: XCTestCase {
    func testPermissionGate() {
        let perms = [CourseCreateLogic.courseCreatePermission]
        XCTAssertTrue(CourseCreateLogic.canCreateCourses(permissions: perms))
        XCTAssertFalse(CourseCreateLogic.canCreateCourses(permissions: ["other"]))
    }

    func testShouldShowNewCourseActionRequiresFlagPermissionAndOnline() {
        var features = MobilePlatformFeatures()
        features.ffMobileCreateCourse = false
        features.ffMobileCourseCreateV2 = false
        let perms = [CourseCreateLogic.courseCreatePermission]
        XCTAssertFalse(
            CourseCreateLogic.shouldShowNewCourseAction(permissions: perms, features: features, isOnline: true)
        )

        features.ffMobileCreateCourse = true
        XCTAssertTrue(
            CourseCreateLogic.shouldShowNewCourseAction(permissions: perms, features: features, isOnline: true)
        )
        XCTAssertFalse(
            CourseCreateLogic.shouldShowNewCourseAction(permissions: perms, features: features, isOnline: false)
        )
        XCTAssertFalse(
            CourseCreateLogic.shouldShowNewCourseAction(permissions: [], features: features, isOnline: true)
        )

        features.ffMobileCreateCourse = false
        features.ffMobileCourseCreateV2 = true
        XCTAssertTrue(
            CourseCreateLogic.shouldShowNewCourseAction(permissions: perms, features: features, isOnline: true)
        )
        XCTAssertTrue(CourseCreateLogic.courseCreateV2Enabled(features))
        XCTAssertEqual(CourseCreateLogic.initialWizardStep(v2Enabled: true), .source)
        XCTAssertEqual(CourseCreateLogic.initialWizardStep(v2Enabled: false), .basics)
    }

    func testValidateCompetenciesParity() {
        XCTAssertEqual(
            CourseCreateLogic.validateCompetencies([])?.key,
            "mobile.createCourse.error.competency.minOne"
        )

        var incomplete = CourseCreateLogic.CompetencyDraft.empty()
        incomplete.title = ""
        XCTAssertEqual(
            CourseCreateLogic.validateCompetencies([incomplete])?.key,
            "mobile.createCourse.error.competency.titleRequired"
        )

        incomplete.title = "Reading"
        incomplete.subOutcomes = []
        XCTAssertEqual(
            CourseCreateLogic.validateCompetencies([incomplete])?.key,
            "mobile.createCourse.error.competency.subOutcomeMinOne"
        )

        incomplete.subOutcomes = [CourseCreateLogic.SubOutcomeDraft(title: "", assessmentTitle: "Quiz 1")]
        XCTAssertEqual(
            CourseCreateLogic.validateCompetencies([incomplete])?.key,
            "mobile.createCourse.error.competency.subOutcomeTitleRequired"
        )

        incomplete.subOutcomes = [CourseCreateLogic.SubOutcomeDraft(title: "Main idea", assessmentTitle: "")]
        XCTAssertEqual(
            CourseCreateLogic.validateCompetencies([incomplete])?.key,
            "mobile.createCourse.error.competency.assessmentTitleRequired"
        )

        let valid = CourseCreateLogic.CompetencyDraft(
            title: "Reading",
            subOutcomes: [
                CourseCreateLogic.SubOutcomeDraft(title: "Main idea", assessmentTitle: "Quiz 1"),
            ]
        )
        XCTAssertNil(CourseCreateLogic.validateCompetencies([valid]))
    }

    func testValidateTitleRequired() {
        XCTAssertNotNil(CourseCreateLogic.validateTitle("  "))
        XCTAssertNil(CourseCreateLogic.validateTitle("Biology 101"))
    }

    func testShouldUpdateExistingCourse() {
        XCTAssertFalse(CourseCreateLogic.shouldUpdateExistingCourse(createdCourseCode: nil))
        XCTAssertFalse(CourseCreateLogic.shouldUpdateExistingCourse(createdCourseCode: "  "))
        XCTAssertTrue(CourseCreateLogic.shouldUpdateExistingCourse(createdCourseCode: "C-ABC123"))
        XCTAssertTrue(CourseCreateLogic.shouldConfirmCancel(createdCourseCode: "C-ABC123"))
    }

    func testSuggestedFirstModuleTitleFromTemplate() {
        XCTAssertEqual(
            CourseCreateLogic.suggestedFirstModuleTitle(templateId: "blank", existing: ""),
            CourseCreateLogic.defaultFirstModuleTitle
        )
        XCTAssertEqual(
            CourseCreateLogic.suggestedFirstModuleTitle(templateId: "k12-semester", existing: ""),
            "Unit 1: Getting started"
        )
        XCTAssertEqual(
            CourseCreateLogic.suggestedFirstModuleTitle(templateId: "k12-semester", existing: "My module"),
            "My module"
        )
    }

    func testTemplateParityWithWeb() {
        let ids = CourseCreateLogic.starterTemplates.map(\.id)
        XCTAssertEqual(ids, [
            "k12-semester",
            "higher-ed-15-week",
            "self-paced",
            "bootcamp",
            "onboarding",
        ])
        XCTAssertEqual(CourseCreateLogic.starterTemplates.count, 5)
        for tmpl in CourseCreateLogic.starterTemplates {
            XCTAssertFalse(tmpl.sections.isEmpty)
            XCTAssertFalse(tmpl.suggestedFirstModuleTitle.isEmpty)
        }
    }

    func testShouldPatchSyllabus() {
        XCTAssertFalse(CourseCreateLogic.shouldPatchSyllabus(templateId: CourseCreateLogic.blankTemplateId))
        XCTAssertTrue(CourseCreateLogic.shouldPatchSyllabus(templateId: "higher-ed-15-week"))
        XCTAssertFalse(CourseCreateLogic.shouldPatchSyllabus(templateId: "unknown"))
    }

    func testTemplateSectionsToSyllabusAssignsIds() {
        let tmpl = CourseCreateLogic.template(for: "self-paced")!
        let sections = CourseCreateLogic.templateSectionsToSyllabus(tmpl.sections)
        XCTAssertEqual(sections.count, tmpl.sections.count)
        XCTAssertEqual(Set(sections.map(\.id)).count, sections.count)
        XCTAssertEqual(sections.first?.heading, tmpl.sections.first?.heading)
    }

    func testBuildCreateRequest() {
        let body = CourseCreateLogic.buildCreateRequest(
            title: "  Chem ",
            description: " Intro ",
            mode: .competencyBased,
            termId: "  ",
            gradeLevel: "9"
        )
        XCTAssertEqual(body.title, "Chem")
        XCTAssertEqual(body.description, "Intro")
        XCTAssertEqual(body.courseType, "competency_based")
        XCTAssertNil(body.termId)
        XCTAssertEqual(body.gradeLevel, "9")
    }

    func testBuildUpdateRequestDoesNotDuplicate() {
        let course = CourseSummary(
            id: "1",
            courseCode: "C-1",
            title: "Old",
            description: "D",
            published: false
        )
        let body = CourseCreateLogic.buildUpdateRequest(
            course: course,
            title: "New title",
            description: "New desc",
            termId: "term-1",
            gradeLevel: "5"
        )
        XCTAssertEqual(body.title, "New title")
        XCTAssertEqual(body.description, "New desc")
        XCTAssertEqual(body.termId, "term-1")
        XCTAssertEqual(body.gradeLevel, "5")
        XCTAssertEqual(body.scheduleMode, "fixed")
        XCTAssertTrue(CourseCreateLogic.shouldUpdateExistingCourse(createdCourseCode: course.courseCode))
    }

    func testModeFromCourseType() {
        XCTAssertEqual(CourseCreateLogic.modeFromCourseType("competency_based"), .competencyBased)
        XCTAssertEqual(CourseCreateLogic.modeFromCourseType("traditional"), .traditional)
        XCTAssertEqual(CourseCreateLogic.modeFromCourseType(nil), .traditional)
    }
}
