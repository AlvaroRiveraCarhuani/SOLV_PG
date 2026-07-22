const fs = require('fs');

const input = fs.readFileSync(0, 'utf-8').trim().split(/\s+/);
if (input.length >= 2) {
    const a = parseInt(input[0], 10);
    const b = parseInt(input[1], 10);
    console.log(a + b);
}
