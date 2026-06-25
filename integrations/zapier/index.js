'use strict';

const authentication = require('./authentication');
const newEnrollment = require('./triggers/new_enrollment');
const gradePosted = require('./triggers/grade_posted');
const assignmentCreated = require('./triggers/assignment_created');
const assignmentSubmitted = require('./triggers/assignment_submitted');
const quizCompleted = require('./triggers/quiz_completed');
const enrollUser = require('./creates/enroll_user');
const createAnnouncement = require('./creates/create_announcement');
const getCourse = require('./creates/get_course');
const getUser = require('./creates/get_user');
const postGrade = require('./creates/post_grade');

module.exports = {
  version: require('./package.json').version,
  platformVersion: require('zapier-platform-core').version,
  authentication,
  triggers: {
    [newEnrollment.key]: newEnrollment,
    [gradePosted.key]: gradePosted,
    [assignmentCreated.key]: assignmentCreated,
    [assignmentSubmitted.key]: assignmentSubmitted,
    [quizCompleted.key]: quizCompleted,
  },
  creates: {
    [enrollUser.key]: enrollUser,
    [createAnnouncement.key]: createAnnouncement,
    [getCourse.key]: getCourse,
    [getUser.key]: getUser,
    [postGrade.key]: postGrade,
  },
};
