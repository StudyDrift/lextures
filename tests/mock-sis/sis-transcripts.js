const express = require('express');
const app = express();
const PORT = 5175;

// A simple GET route for the home page
app.use(express.json());

app.post('/', (req, res) => {
    console.log('Request body:', JSON.stringify(req.body, null, 2));
    res.status(200).send('Everything is great!');
});

app.get('/', (req, res) => {
    res.status(200).send('Test');
});

// Start the server and listen on the specified port
app.listen(PORT, () => {
    console.log(`Server is cruising along at http://localhost:${PORT}`);
});
