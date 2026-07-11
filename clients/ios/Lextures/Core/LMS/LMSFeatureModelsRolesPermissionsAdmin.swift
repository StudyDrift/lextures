import Foundation

// MARK: - Roles & permissions admin (M14.2)

struct RBACPermission: Decodable, Identifiable, Hashable {
    var id: String
    var permissionString: String
    var description: String
    var createdAt: String?
}

struct RoleWithPermissions: Decodable, Identifiable, Hashable {
    var id: String
    var name: String
    var description: String?
    var scope: String?
    var createdAt: String?
    var permissions: [RBACPermission]
}

struct RBACUserBrief: Decodable, Identifiable, Hashable {
    var id: String
    var email: String
    var displayName: String?
    var sid: String?
}

struct RolesListResponse: Decodable {
    var roles: [RoleWithPermissions]
}

struct RoleUsersResponse: Decodable {
    var users: [RBACUserBrief]
}

struct AddRoleUserRequest: Encodable {
    var userId: String
}
