package com.lextures.android.core.lms

object CurrencyExponent {
    private val zeroDecimalCurrencies = setOf("jpy")

    fun isZeroDecimal(currency: String): Boolean =
        zeroDecimalCurrencies.contains(currency.lowercase().trim())

    fun minorUnitFactor(currency: String): Int = if (isZeroDecimal(currency)) 1 else 100

    fun minorUnitsToMajorUnits(minor: Int, currency: String): Double =
        minor.toDouble() / minorUnitFactor(currency)

    fun majorUnitsToMinorUnits(major: Double, currency: String): Int =
        Math.round(major * minorUnitFactor(currency)).toInt()

    fun maxCatalogMinorUnits(currency: String): Int =
        if (isZeroDecimal(currency)) 99_999 else 9_999_999

    const val MAX_PRICE_MAJOR_ZERO_DECIMAL = 99_999.0
}