package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class BillingLogicTest {
    @Test
    fun billingEnabledRequiresFeatureFlag() {
        val off = MobilePlatformFeatures()
        assertFalse(BillingLogic.billingEnabled(off))
        assertTrue(BillingLogic.billingEnabled(off.copy(ffStripeBilling = true)))
        assertTrue(BillingLogic.billingEnabled(off.copy(ffPaymentsEnabled = true)))
    }

    @Test
    fun checkoutUrlsIncludeCourseId() {
        assertTrue(BillingLogic.checkoutSuccessPath("abc-123").contains("course_id=abc-123"))
        assertEquals("/checkout/cancel", BillingLogic.checkoutCancelPath())
    }

    @Test
    fun checkoutEndpointSelection() {
        assertEquals("/api/v1/checkout", BillingLogic.checkoutEndpoint(usePaymentsAbstraction = true))
        assertEquals("/api/v1/billing/checkout", BillingLogic.checkoutEndpoint(usePaymentsAbstraction = false))
    }

    @Test
    fun subscriptionDetection() {
        assertTrue(BillingLogic.isSubscriptionEntitlement("subscription_annual"))
        assertFalse(BillingLogic.isSubscriptionEntitlement("course_purchase"))
    }
}