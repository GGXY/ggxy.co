const fastify = require('fastify')({
    logger: true
});

fastify.register(require('./paths.js'));
fastify.register(require('./addData.js'));
fastify.setNotFoundHandler(function (request, reply) { // same of `setErrorHandler`
    reply.redirect(301, "https://gamergalaxy.net");
});

fastify.get('/', async (request, reply) => {
    reply.redirect(301, "https://gamergalaxy.net");
});

const redirectDirectPath = async(request, reply) => {
    reply.redirect(301, "https://gamergalaxy.net" + request.url);
};
fastify.get('/games', redirectDirectPath);
fastify.get('/games/:slug', redirectDirectPath);
fastify.get('/news', redirectDirectPath);
fastify.get('/news/:slug', redirectDirectPath);



const start = async () => {
    try {
        await fastify.listen(3003, '0.0.0.0')
    } catch (err) {
        fastify.log.error(err)
        process.exit(1)
    }
}
start();