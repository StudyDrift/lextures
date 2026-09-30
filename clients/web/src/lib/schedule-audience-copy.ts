export type ScheduleAudienceCopy = {
  sectionTitle: string
  sectionIntro: string
  relativeSwitchLabel: string
  fixedSwitchLabel: string
  modeLabelRelative: string
  modeLabelFixed: string
  relativeBody: string
}

const campus: ScheduleAudienceCopy = {
  sectionTitle: 'Fixed Schedule & Visibility',
  sectionIntro:
    'Control whether the course uses fixed calendar dates or a timeline from each student’s enrollment. Module release and due dates follow the same mode: relative courses shift those dates by the same offset.',
  relativeSwitchLabel: 'Relative schedule from each enrollment',
  fixedSwitchLabel: 'Fixed calendar schedule (not relative to enrollment)',
  modeLabelRelative: 'Relative (from enrollment)',
  modeLabelFixed: 'Fixed (calendar dates)',
  relativeBody:
    'Start and catalog visibility begin when the student is enrolled. Set how long the course runs and when it drops from the catalog (optional). Durations use ISO-style lengths (days, weeks, months, or years).',
}

const family: ScheduleAudienceCopy = {
  sectionTitle: 'Family schedule & visibility',
  sectionIntro:
    'Choose fixed calendar dates or a family schedule that starts when each learner begins. Module release and due dates follow the same mode, so a flexible pace shifts those dates together.',
  relativeSwitchLabel: 'Family schedule from when each learner starts',
  fixedSwitchLabel: 'Fixed calendar schedule',
  modeLabelRelative: 'Flexible (from when they start)',
  modeLabelFixed: 'Fixed (calendar dates)',
  relativeBody:
    'The course becomes visible when the learner starts. Set how long it runs and when it leaves the catalog (optional). Durations use days, weeks, months, or years.',
}

export function scheduleAudienceCopy(familyAudience: boolean): ScheduleAudienceCopy {
  return familyAudience ? family : campus
}
