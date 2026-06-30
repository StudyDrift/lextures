import XCTest
@testable import Lextures

final class SettingsAccommodationsTests: XCTestCase {
    // MARK: - Account profile

    func testDecodesAccountProfile() throws {
        let json = """
        {"email":"ada@example.com","displayName":"Ada Lovelace","firstName":"Ada",\
        "lastName":"Lovelace","avatarUrl":"https://img/avatar.png","phoneNumber":"+1 555 0100",\
        "uiTheme":"dark","locale":"en"}
        """
        let profile = try JSONDecoder().decode(AccountProfile.self, from: Data(json.utf8))
        XCTAssertEqual(profile.email, "ada@example.com")
        XCTAssertEqual(profile.firstName, "Ada")
        XCTAssertEqual(profile.lastName, "Lovelace")
        XCTAssertEqual(profile.avatarUrl, "https://img/avatar.png")
        XCTAssertEqual(profile.phoneNumber, "+1 555 0100")
    }

    func testAccountProfileToleratesMissingFields() throws {
        let json = """
        {"email":"only@example.com"}
        """
        let profile = try JSONDecoder().decode(AccountProfile.self, from: Data(json.utf8))
        XCTAssertEqual(profile.email, "only@example.com")
        XCTAssertNil(profile.firstName)
        XCTAssertNil(profile.phoneNumber)
    }

    func testAccountProfileResolvesNameFieldsFromDisplayName() throws {
        let json = """
        {"email":"ada@example.com","displayName":"Ada Lovelace","avatarUrl":"https://img/a.png"}
        """
        let profile = try JSONDecoder().decode(AccountProfile.self, from: Data(json.utf8))
        XCTAssertEqual(profile.resolvedNameFields.firstName, "Ada")
        XCTAssertEqual(profile.resolvedNameFields.lastName, "Lovelace")
        XCTAssertEqual(profile.resolvedDisplayName, "Ada Lovelace")
        XCTAssertEqual(profile.resolvedInitials, "AL")
    }

    func testAccountPatchEncodesEditableFields() throws {
        let patch = AccountProfilePatch(
            firstName: "Grace",
            lastName: "Hopper",
            avatarUrl: "",
            phoneNumber: "555"
        )
        let data = try JSONEncoder().encode(patch)
        let object = try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
        XCTAssertEqual(object["firstName"] as? String, "Grace")
        XCTAssertEqual(object["lastName"] as? String, "Hopper")
        XCTAssertEqual(object["avatarUrl"] as? String, "")
        XCTAssertEqual(object["phoneNumber"] as? String, "555")
    }

    // MARK: - Accommodations

    func testDecodesMyAccommodations() throws {
        let json = """
        {"accommodations":[
          {"courseCode":"MATH101","hasExtendedTime":true,"hasExtraAttempts":false,
           "hintsAlwaysAvailable":true,"reducedDistractionRecommended":false,
           "speechToTextEnabled":false,"ttsEnabled":true,"dyslexiaDisplayEnabled":false,
           "highContrastEnabled":false,"reducedMotionEnabled":false,"separateSetting":false,
           "effectiveFrom":"2026-01-01","effectiveUntil":"2026-12-31"},
          {"hasExtendedTime":false,"hasExtraAttempts":false,"hintsAlwaysAvailable":false,
           "reducedDistractionRecommended":false,"speechToTextEnabled":false,"ttsEnabled":false,
           "dyslexiaDisplayEnabled":false,"highContrastEnabled":false,"reducedMotionEnabled":false,
           "separateSetting":false}
        ]}
        """
        let response = try JSONDecoder().decode(MyAccommodationsResponse.self, from: Data(json.utf8))
        XCTAssertEqual(response.accommodations.count, 2)

        let first = response.accommodations[0]
        XCTAssertEqual(first.courseCode, "MATH101")
        XCTAssertEqual(first.id, "MATH101")
        XCTAssertTrue(first.hasExtendedTime)
        XCTAssertTrue(first.ttsEnabled)
        XCTAssertFalse(first.isEmpty)
        XCTAssertEqual(first.effectiveFrom, "2026-01-01")

        let second = response.accommodations[1]
        XCTAssertNil(second.courseCode)
        XCTAssertEqual(second.id, "__all__")
        XCTAssertTrue(second.isEmpty)
    }

    // MARK: - Theme preference

    func testThemeAppearanceMapsToColorScheme() {
        XCTAssertNil(ThemePreference.Appearance.system.colorScheme)
        XCTAssertEqual(ThemePreference.Appearance.light.colorScheme, .light)
        XCTAssertEqual(ThemePreference.Appearance.dark.colorScheme, .dark)
    }

    func testThemePreferencePersistsSelection() {
        let preference = ThemePreference.shared
        let original = preference.appearance
        defer { preference.appearance = original }

        preference.appearance = .dark
        XCTAssertEqual(UserDefaults.standard.string(forKey: "lextures.theme.appearance"), "dark")
        preference.appearance = .light
        XCTAssertEqual(UserDefaults.standard.string(forKey: "lextures.theme.appearance"), "light")
    }
}
