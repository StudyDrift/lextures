import Foundation

extension LMSAPI {
    static func fetchPeerReviewAssigned(accessToken: String) async throws -> [PeerReviewAllocation] {
        let (data, response) = try await client.request(
            path: "/api/v1/peer-review/assigned",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            return []
        }
        return try decode(PeerReviewAssignedResponse.self, from: data).allocations
    }

    static func fetchPeerReviewAllocation(
        allocationId: String,
        accessToken: String
    ) async throws -> PeerReviewAllocationDetail {
        let (data, _) = try await client.request(
            path: "/api/v1/peer-review/allocations/\(encodePath(allocationId))",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(PeerReviewAllocationDetail.self, from: data)
    }

    static func submitPeerReview(
        allocationId: String,
        body: PeerReviewSubmitRequest,
        accessToken: String
    ) async throws -> PeerReviewSubmitResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/peer-review/allocations/\(encodePath(allocationId))",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(PeerReviewSubmitResponse.self, from: data)
    }

    static func fetchPeerReviewReceived(
        courseCode: String,
        assignmentId: String,
        accessToken: String
    ) async throws -> [PeerReviewReceivedItem] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/assignments/\(encodePath(assignmentId))/peer-review/received",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            return []
        }
        return try decode(PeerReviewReceivedResponse.self, from: data).reviews
    }
}
