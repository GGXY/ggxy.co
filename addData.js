const data = require("./data/db.json");
const fs = require('fs');
const util = require('util');
const writeFile = (fileName, data) => util.promisify(fs.writeFile)(fileName, data);
const random = (length = 8) => {
    return Math.random().toString(16).substr(2, length);
};

const keyNeeded = process.env.SECRET || random(16);

if (!process.env.SECRET) {
    console.log("Secret key is: ", keyNeeded);
}

async function routes (fastify, options) {
    fastify.post("/update", async (request, reply) => {
        
        if (request.headers.authorization != "Bearer " + keyNeeded) {
            reply.code(403);
            reply.send({status:"failed"});
            return;
        }
        
        var newRoute = request.body;
        if (!newRoute || !newRoute.path || !newRoute.redirect) {
            reply.code(400);
            reply.send({status:"failed"});
            return;
        }

        const indx = data.findIndex(route => route.path == newRoute.path);
        if (indx !== -1) {
            data.splice(indx, 1);
        }
        data.push({
            path: newRoute.path,
            redirect: newRoute.redirect
        });

        await writeFile("./data/db.json", JSON.stringify(data, undefined, 4));

        reply.send({status:"success"});
    });

    fastify.delete("/update", async (request, reply) => {
        if (request.headers.authorization != "Bearer " + keyNeeded) {
            reply.code(403);
            reply.send({status:"failed"});
            return;
        }
        
        var newRoute = request.body;
        if (!newRoute || !newRoute.path) {
            reply.code(400);
            reply.send({status:"failed"});
            return;
        }
        const indx = data.findIndex(route => route.path == newRoute.path);
        if (indx !== -1) {
            data.splice(indx, 1);
        }
        await writeFile("./data/db.json", JSON.stringify(data, undefined
            , 4));
        reply.send({status:"success"});
    });
}
    
module.exports = routes;
    