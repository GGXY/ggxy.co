FROM node:12-alpine

WORKDIR /app

COPY  ./addData.js ./addData.js
COPY  ./index.js ./index.js
COPY  ./paths.js ./paths.js
VOLUME /app/data

CMD [ "node", "index.js" ]