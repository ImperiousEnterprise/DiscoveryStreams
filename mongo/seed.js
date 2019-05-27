var fs=require('fs');
var data=fs.readFileSync('streams.json', 'utf8');
var words=JSON.parse(data);

db.test.drop();
db.test.insertMany(words);