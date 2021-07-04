const data = require("./data/db.json");

async function routes (fastify, options) {
    for(const route of data) {
        fastify.get('/' + route.path, async (request, reply) => {
            reply.redirect(301, route.redirect);
        });
    }
}
  
module.exports = routes;
