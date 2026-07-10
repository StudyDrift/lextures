/** Whether to show the purchased badge for a course (feature flag + entitlement). */
export function shouldShowPurchasedBadge(
  ffCourseMarketplace: boolean | undefined,
  course: { acquiredViaMarketplace?: boolean },
): boolean {
  return ffCourseMarketplace === true && course.acquiredViaMarketplace === true
}
