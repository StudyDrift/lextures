import Foundation

// MARK: - Org structure admin (M14.4)

struct AdminOrgRow: Decodable, Identifiable, Hashable {
    var id: String
    var slug: String
    var name: String
    var status: String
    var dataRegion: String?
    var userCount: Int64?
    var courseCount: Int64?
    var createdAt: String?
}

struct AdminOrgsListResponse: Decodable {
    var organizations: [AdminOrgRow]?
}

struct OrgUnitTreeNode: Decodable, Identifiable, Hashable {
    var id: String
    var name: String
    var unitType: String
    var status: String
    var childCourseCount: Int64?
    var children: [OrgUnitTreeNode]?
}

struct OrgUnitTreeResponse: Decodable {
    var tree: [OrgUnitTreeNode]?
}

struct CreateAcademicTermRequest: Encodable {
    var name: String
    var termType: String
    var startDate: String
    var endDate: String
}

struct PatchAcademicTermRequest: Encodable {
    var name: String?
    var termType: String?
    var startDate: String?
    var endDate: String?
    var status: String?
}

struct PatchOrgUnitRequest: Encodable {
    var name: String?
}
