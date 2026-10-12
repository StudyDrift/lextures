package starterlibrary

import "strings"

func catalog() []Course {
	seeds := []seed{
		{slug: "math-k-2", title: "Numbers and patterns", band: BandK2, subject: SubjectMath, minutes: 40,
			standards: []string{"CCSS.MATH.CONTENT.K.CC.B.4", "CCSS.MATH.CONTENT.1.OA.A.1"},
			summary:   "Count, compare, and notice patterns with objects at home.",
			goal:      "Count a set and say how many without starting over",
			detail:    "Put six buttons in a line. Touch each one as you count: one, two, three, four, five, six. The last number tells how many. Move the buttons into a circle and count again. The number stays six.",
			talk:      "Which group has more: four spoons or two cups? How do you know?",
			try:       "Find 10 small objects. Make two groups that are the same size, then make groups of 2.",
			check:     "You have 5 crackers and eat 2. How many are left?",
			unit:      "Counting that sticks", practice: "Make equal groups"},
		{slug: "math-3-5", title: "Multiplication and fractions", band: Band35, subject: SubjectMath, minutes: 50,
			standards: []string{"CCSS.MATH.CONTENT.3.OA.A.1", "CCSS.MATH.CONTENT.4.NF.A.1"},
			summary:   "See multiplication as equal groups, then name parts of a whole.",
			goal:      "Explain a multiplication fact as groups, and a fraction as equal parts",
			detail:    "3 × 4 means three groups of four. Draw it. A fraction names equal parts: 1/2 of a sandwich is one of two same-sized pieces. 2/4 of the same sandwich is two smaller pieces, and it is the same amount as 1/2.",
			talk:      "Is 1/2 of a small cracker the same amount as 1/2 of a large apple? What is the same, and what is different?",
			try:       "Fold a paper into fourths. Shade 1/2 one way and 2/4 another way. Write a sentence about what you see.",
			check:     "Draw 4 × 3 as groups, then write the product.",
			unit:      "Equal groups", practice: "Fractions of a whole"},
		{slug: "math-6-8", title: "Ratios, rates, and expressions", band: Band68, subject: SubjectMath, minutes: 60,
			standards: []string{"CCSS.MATH.CONTENT.6.RP.A.1", "CCSS.MATH.CONTENT.7.EE.B.4"},
			summary:   "Compare quantities with ratios and write a simple equation for an unknown.",
			goal:      "Write a ratio, a unit rate, and an equation with one unknown",
			detail:    "A recipe uses 2 cups of water for every 1 cup of rice. The ratio of water to rice is 2:1. If you cook 3 cups of rice, you need 6 cups of water. An expression like 2r names the water when r is the cups of rice.",
			talk:      "A store sells 3 pencils for $1.50. What is the price of one pencil, and how did you get it?",
			try:       "Time a walk for 2 minutes and count your steps. Estimate steps for 10 minutes and show the rate you used.",
			check:     "Solve 2x + 4 = 16 and say what x means in a sentence.",
			unit:      "Ratios and rates", practice: "Equations for an unknown"},
		{slug: "math-9-12", title: "Linear functions and models", band: Band912, subject: SubjectMath, minutes: 75,
			standards: []string{"CCSS.MATH.CONTENT.HSA.CED.A.2", "CCSS.MATH.CONTENT.HSF.IF.B.4"},
			summary:   "Model a steady change with a table, a graph, and y = mx + b.",
			goal:      "Build a linear model from a situation and read its slope and intercept",
			detail:    "A ride costs $3 to start and $2 each mile. After m miles the cost is C = 2m + 3. The slope 2 is the change in cost per mile. The intercept 3 is the cost at zero miles. A graph of the line rises by 2 for every 1 step right.",
			talk:      "When would a linear model be a poor fit, such as a phone battery or a population that levels off?",
			try:       "Pick a bill or a savings plan with a start amount and a steady weekly change. Write the equation, make a four-row table, and sketch the line.",
			check:     "For C = 2m + 3, what is the cost at 0 miles and at 5 miles?",
			unit:      "Slope and intercept", practice: "Model a situation"},
		{slug: "ela-k-2", title: "Sounds, stories, and sentences", band: BandK2, subject: SubjectELA, minutes: 40,
			standards: []string{"CCSS.ELA-LITERACY.RF.K.2", "CCSS.ELA-LITERACY.RL.1.1"},
			summary:   "Hear sounds in words, retell a short story, and write one clear sentence.",
			goal:      "Retell who and what happened, then write a sentence about it",
			detail:    "Say the word map. Stretch it: /m/ /a/ /p/. Those are the sounds. After a read-aloud, name the character and one thing they did. A sentence starts with a capital letter, tells a complete idea, and ends with a period.",
			talk:      "What happened first, next, and last in the story you read today?",
			try:       "Read or listen to a picture book. Draw three boxes for beginning, middle, and end. Write or dictate one sentence for the middle.",
			check:     "Which sentence is complete: \"The cat.\" or \"The cat sat.\"? Why?",
			unit:      "Sounds and retelling", practice: "One strong sentence"},
		{slug: "ela-3-5", title: "Reading for the main idea", band: Band35, subject: SubjectELA, minutes: 50,
			standards: []string{"CCSS.ELA-LITERACY.RI.4.2", "CCSS.ELA-LITERACY.W.4.1"},
			summary:   "Find the main idea, point to a detail that supports it, and write a short opinion.",
			goal:      "State a main idea in your own words and back it with a detail",
			detail:    "The main idea is what the whole passage is mostly about, not a single fact. A detail proves it. An opinion needs a reason: \"I think we should read outside because the porch is quiet after lunch.\"",
			talk:      "Tell the main idea of a page you read this week without copying the first sentence.",
			try:       "Read a short article or a section of a book. Write the main idea in one sentence and copy or paraphrase two details under it.",
			check:     "What is the difference between a topic (dogs) and a main idea (working dogs can be trained to help people)?",
			unit:      "Main idea and details", practice: "A short opinion"},
		{slug: "ela-6-8", title: "Evidence in what you read", band: Band68, subject: SubjectELA, minutes: 60,
			standards: []string{"CCSS.ELA-LITERACY.RI.7.1", "CCSS.ELA-LITERACY.W.7.1"},
			summary:   "Quote or paraphrase evidence and explain how it supports a claim.",
			goal:      "Make a claim and tie it to a cited detail from a text",
			detail:    "A claim is a point you can defend. Evidence is a fact, quote, or example from the text. The explanation is the sentence that shows how the evidence proves the claim. \"The author says the bridge closed\" is evidence. \"That shows the storm was severe enough to stop travel\" is the explanation.",
			talk:      "Find one sentence in a news story that is a fact and one that is the writer's judgment.",
			try:       "Read two pages. Write a claim about the author's point, quote a short phrase, and add one sentence of explanation.",
			check:     "Why is a quote alone not enough in a paragraph of argument?",
			unit:      "Claim and evidence", practice: "Explain the quote"},
		{slug: "ela-9-12", title: "Argument and analysis", band: Band912, subject: SubjectELA, minutes: 75,
			standards: []string{"CCSS.ELA-LITERACY.RI.9-10.1", "CCSS.ELA-LITERACY.W.9-10.1"},
			summary:   "Analyze how a writer builds an argument and draft a response with a counterpoint.",
			goal:      "Trace a writer's claim, evidence, and a limitation, then answer it",
			detail:    "Strong analysis names the claim, the best evidence, and what the writer leaves out. A counterpoint is a fair objection, not a straw man. Your response can agree in part and still add a limit: the claim holds when this condition is true, and weakens when it is not.",
			talk:      "What would someone who disagrees say, and which sentence in the text would they point to?",
			try:       "Choose an editorial, speech, or textbook section. Outline claim, two pieces of evidence, one limitation, and your response in four sentences.",
			check:     "What makes a counterpoint fair instead of a dismissal?",
			unit:      "How an argument is built", practice: "Write the response"},
		{slug: "science-k-2", title: "Plants, weather, and pushes", band: BandK2, subject: SubjectScience, minutes: 40,
			standards: []string{"1-LS1-1", "K-ESS2-1"},
			summary:   "Watch living things, local weather, and how a push or pull changes motion.",
			goal:      "Record an observation and say what changed",
			detail:    "A plant needs light and water. You can see new growth over days, not minutes. Weather is what the air is doing today: sun, cloud, rain, wind. A push starts a toy car. A stronger push makes it go farther on the same floor.",
			talk:      "What was the weather at breakfast, and what is it now?",
			try:       "Pick a plant or a patch of sky. Draw it today and again in two days. Roll a ball with a gentle push and a stronger push and mark where it stops.",
			check:     "Name one thing a plant needs and one way a push changes a ball.",
			unit:      "Look and record", practice: "Pushes and weather"},
		{slug: "science-3-5", title: "Ecosystems and energy", band: Band35, subject: SubjectScience, minutes: 50,
			standards: []string{"3-LS4-3", "4-PS3-2"},
			summary:   "Trace how living things depend on each other and how energy moves.",
			goal:      "Describe a food link and a transfer of energy",
			detail:    "In a habitat, plants capture light and animals eat plants or other animals. Change one part, such as fewer insects, and birds that eat them have less food. Energy can move: a moving marble can knock another marble, and the motion transfers.",
			talk:      "Name a plant, an animal that eats it, and an animal that eats that animal, from a place you know.",
			try:       "Draw a three-step food chain for a yard, park, or book habitat. Then drop a ball onto a ruler balanced on a pencil and describe what moves.",
			check:     "What happens to an animal if the plant it depends on disappears?",
			unit:      "Who eats whom", practice: "Energy on the move"},
		{slug: "science-6-8", title: "Matter, motion, and Earth systems", band: Band68, subject: SubjectScience, minutes: 60,
			standards: []string{"MS-PS1-1", "MS-ESS2-4"},
			summary:   "Model particles, describe motion with forces, and trace water through Earth systems.",
			goal:      "Use a model to explain a change you can observe",
			detail:    "Matter is made of particles too small to see. In a solid they stay close; in a gas they move apart. A force can speed an object up, slow it down, or change its direction. Water evaporates, condenses into clouds, and returns as precipitation. That cycle moves water among ocean, air, and land.",
			talk:      "Where does the water on a cold glass come from if you did not spill any?",
			try:       "Warm a sealed bag with a damp paper towel in the sun and a matching bag in the shade. Record what you see after 20 minutes and explain it with particles or the water cycle.",
			check:     "Name the force that slows a rolling ball on carpet, and one step of the water cycle.",
			unit:      "Particles and forces", practice: "Water on the move"},
		{slug: "science-9-12", title: "Cells, forces, and climate evidence", band: Band912, subject: SubjectScience, minutes: 75,
			standards: []string{"HS-LS1-1", "HS-ESS3-5"},
			summary:   "Connect cell structures to function and read evidence about climate.",
			goal:      "Explain a structure-function pair and distinguish data from a conclusion",
			detail:    "A cell membrane controls what enters and leaves. Mitochondria release usable energy from food. A claim about climate rests on measurements over time, such as temperature records and ice cores, not on one hot afternoon. A model predicts, and new measurements test the prediction.",
			talk:      "Which cell part would you expect to be abundant in a muscle cell, and why?",
			try:       "Sketch an animal cell, label three parts, and write the job of each. Then list two kinds of evidence scientists use when they describe a warming climate, and one limit of a short personal observation.",
			check:     "Why is one week's weather not enough to describe climate?",
			unit:      "Cell structure and function", practice: "Evidence and claims"},
		{slug: "history-k-2", title: "Families, communities, and maps", band: BandK2, subject: SubjectHistory, minutes: 40,
			standards: []string{"D2.Civ.2.K-2", "D2.Geo.1.K-2"},
			summary:   "Place your family in a community and read a simple map.",
			goal:      "Tell who helps in a community and use a map word such as near or far",
			detail:    "A community is the people and places around you: home, library, store, park. Maps use symbols. A key tells what a symbol means. Near and far, left and right, help someone else find a place.",
			talk:      "Who is one helper in your community, and what do they do?",
			try:       "Draw a map from your door to one other room or from your home to a nearby place. Add a key with two symbols.",
			check:     "What does a map key do?",
			unit:      "People and places", practice: "Make a map"},
		{slug: "history-3-5", title: "Communities then and now", band: Band35, subject: SubjectHistory, minutes: 50,
			standards: []string{"D2.His.2.3-5", "D2.Civ.1.3-5"},
			summary:   "Compare daily life across time and name a rule a community uses.",
			goal:      "Use a source to compare then and now, and explain one community rule",
			detail:    "Historians ask what changed and what stayed the same. A photograph, a letter, or an object is a source. Communities use rules so people can share roads, schools, and parks. A rule works better when people know why it exists.",
			talk:      "What is one tool your grandparents used that your home uses differently now?",
			try:       "Pick a chore or a trip to school. List two ways it might have looked 50 years ago and two ways it looks now. Name the source of your guess, even if the source is a person you asked.",
			check:     "What is the difference between a guess and a source?",
			unit:      "Change and continuity", practice: "Rules we share"},
		{slug: "history-6-8", title: "Worlds in contact", band: Band68, subject: SubjectHistory, minutes: 60,
			standards: []string{"D2.His.1.6-8", "D2.Geo.2.6-8"},
			summary:   "Study how trade and movement connected people, and whose perspective a source leaves out.",
			goal:      "Explain one exchange between regions and name a missing perspective",
			detail:    "Trade moves goods and ideas. When regions meet, both sides change, and the benefits are rarely equal. A map drawn by one empire shows what that empire cared about. Asking who is not quoted is part of reading history.",
			talk:      "Name a food, word, or tool in your house that came from somewhere else. How might it have traveled?",
			try:       "Choose one exchange, such as silk, horses, or printed books. Write who gained, who paid a cost, and one group the short account does not describe.",
			check:     "Why can two accurate sources still tell different stories about the same meeting of peoples?",
			unit:      "Exchange", practice: "Perspective"},
		{slug: "history-9-12", title: "Modern world history", band: Band912, subject: SubjectHistory, minutes: 75,
			standards: []string{"D2.His.1.9-12", "D2.Civ.5.9-12"},
			summary:   "Examine a modern turning point with causes, evidence, and civic consequences.",
			goal:      "Separate causes from a single event and connect the event to a civic question",
			detail:    "A turning point has causes that built up and effects that lasted. Industrialization, war, independence movements, and civil rights campaigns are not one speech or one battle. Civic life asks who could vote, speak, or be protected by law before and after the change.",
			talk:      "Pick a 20th-century event you know. What is one cause that started before the day people remember?",
			try:       "Outline a turning point in four lines: context, trigger, two effects, and one civic question still open. Cite the kind of source you would trust for each effect.",
			check:     "Why is the trigger of an event not the same thing as its causes?",
			unit:      "Causes of a turning point", practice: "Civic consequences"},
		{slug: "art-k-2", title: "Shapes and stories", band: BandK2, subject: SubjectElective, minutes: 35,
			standards: []string{"VA:Cr1.1.Ka"},
			summary:   "Make a picture that tells a small story with shapes and color.",
			goal:      "Choose shapes on purpose and say what they show",
			detail:    "A circle can be a sun, a wheel, or a face. Color can show feeling: a bright yellow sun or a dark blue night. Artists decide, then look again and change one thing.",
			talk:      "What is the most important shape in your picture, and why did you pick it?",
			try:       "Draw a place you know using only circles, squares, and triangles. Tell a one-sentence story about the picture.",
			check:     "Name two shapes you used and what each one stands for.",
			unit:      "Shapes with a job", practice: "A picture story"},
		{slug: "music-3-5", title: "Rhythm and listening", band: Band35, subject: SubjectElective, minutes: 40,
			standards: []string{"MU:Pr4.2.5a"},
			summary:   "Keep a steady beat, hear a pattern, and describe a piece of music.",
			goal:      "Tap a steady beat and name one thing you hear",
			detail:    "Beat is the steady pulse. Rhythm is the pattern of long and short sounds on that pulse. Loud and soft, fast and slow, and the instruments you recognize are all fair things to notice.",
			talk:      "Clap a pattern of four beats. Can someone else clap it back?",
			try:       "Listen to one minute of a song. Tap the beat, then write three words about what you heard, such as fast, drums, or quiet.",
			check:     "What is the difference between the beat and the rhythm?",
			unit:      "Beat and pattern", practice: "Listen on purpose"},
		{slug: "computing-6-8", title: "How programs think", band: Band68, subject: SubjectElective, minutes: 55,
			standards: []string{"2-AP-11"},
			summary:   "Break a task into steps a computer could follow, including a decision.",
			goal:      "Write a short sequence with a condition and test it",
			detail:    "A program is a precise list of steps. Sequence means the order matters. A condition is an if: if the cup is empty, fill it; else drink. Bugs are steps that do not match what you meant. Testing finds them.",
			talk:      "Where would a missing step break a recipe you know?",
			try:       "Write steps for making a snack that include one if. Hand them to someone else and change any step they cannot follow.",
			check:     "What is a condition, and why does order matter in a sequence?",
			unit:      "Steps and decisions", practice: "Test the steps"},
		{slug: "money-9-12", title: "Money choices", band: Band912, subject: SubjectElective, minutes: 60,
			standards: []string{"Jump$tart Spending and Saving"},
			summary:   "Track money in and out, tell needs from wants, and set a short-term goal.",
			goal:      "Build a simple plan that names income, spending, and a goal",
			detail:    "Income is money coming in. An expense is money going out. A need is required for health and safety. A want can wait. A goal with an amount and a date is easier to check than \"save more.\"",
			talk:      "Name one need and one want from the last week, and what each cost if you know.",
			try:       "List one week of money in and money out, even if the numbers are small or estimated. Circle one change that would move you toward a goal you write down.",
			check:     "Why does a savings goal need both an amount and a date?",
			unit:      "In, out, and goals", practice: "A one-week plan"},
	}
	out := make([]Course, 0, len(seeds))
	for _, s := range seeds {
		out = append(out, s.course())
	}
	return out
}

// Filter returns courses matching optional band and subject. Empty filters match all.
func Filter(band, subject string) []Course {
	band = strings.TrimSpace(band)
	subject = strings.TrimSpace(subject)
	all := catalog()
	if band == "" && subject == "" {
		return all
	}
	out := make([]Course, 0, len(all))
	for _, c := range all {
		if band != "" && c.GradeBand != band {
			continue
		}
		if subject != "" && c.Subject != subject {
			continue
		}
		out = append(out, c)
	}
	return out
}

// Get returns a starter course by slug.
func Get(slug string) (Course, bool) {
	slug = strings.TrimSpace(slug)
	for _, c := range catalog() {
		if c.Slug == slug {
			return c, true
		}
	}
	return Course{}, false
}
