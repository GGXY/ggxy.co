FROM node:12-alpine

WORKDIR /app

COPY  ./addData.js ./addData.js
COPY  ./index.js ./index.js
COPY  ./paths.js ./paths.js
COPY  ./package.json ./package.json
VOLUME /app/data

RUN ["npm", "install"]
CMD [ "node", "index.js" ]