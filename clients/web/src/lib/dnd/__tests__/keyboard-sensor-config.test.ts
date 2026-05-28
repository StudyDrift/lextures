import { describe, expect, it } from 'vitest'
import { KeyboardSensor, defaultKeyboardSensorOptions } from '../keyboardSensorConfig'

describe('keyboardSensorConfig', () => {
  it('exports KeyboardSensor class', () => {
    expect(KeyboardSensor).toBeDefined()
    expect(typeof KeyboardSensor).toBe('function')
  })

  it('exports defaultKeyboardSensorOptions with a coordinateGetter', () => {
    expect(defaultKeyboardSensorOptions).toHaveProperty('coordinateGetter')
    expect(typeof defaultKeyboardSensorOptions.coordinateGetter).toBe('function')
  })

  it('coordinateGetter is stable (same reference each import)', async () => {
    const { defaultKeyboardSensorOptions: opts2 } = await import('../keyboardSensorConfig')
    expect(defaultKeyboardSensorOptions.coordinateGetter).toBe(opts2.coordinateGetter)
  })
})
