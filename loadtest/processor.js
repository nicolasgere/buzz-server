

const MESSAGES = [
    'what a nice day',
    'how\'s everybody?',
    'how\'s it going?',
    'what a lovely socket.io chatroom',
    'to be or not to be, that is the question',
    'Romeo, Romeo! wherefore art thou Romeo?',
    'now is the winter of our discontent.',
    'get thee to a nunnery',
    'a horse! a horse! my kingdom for a horse!'
];

const topic_number = 100


function initWs(context, events, done) {
    var index = Math.floor(Math.random() * MESSAGES.length)
    var username =  "user-" + Math.floor(Math.random() * 100000).toString()
    context.vars.username = username
    var m = {username:username, message:MESSAGES[index]};
    context.vars.message = Buffer.from(JSON.stringify(m)).toString('base64')
    context.vars.topic =  "topic-" + Math.floor(Math.random() * topic_number).toString()
    var h = {username:username, writing:false};
    context.vars.heartbeat = JSON.stringify(h)
    console.log(context.vars.topic)
    return done();
}
module.exports = {
    initWs,
}