const http = require('node:http');
const https = require('node:https');
const querystring = require('node:querystring');

const hostname = 'localhost';
const port = 3000;

// Start the server and open the following URL in a web browser:
// ...
const server = http.createServer((req, res) => {
  res.statusCode = 200;
  res.setHeader('Content-Type', 'text/plain');
  const params = querystring.decode(req.url.split('?')[1]);
  const code = params['code']
  if (code) {
    const data = 'grant_type=authorization_code'
      + '&code=' + code
      + '&redirect_uri=http://' + hostname + ':' + port; 

    const options = {
      hostname: 'api.pinterest.com',
      port: 443,
      path: '/v5/oauth/token',
      method: 'POST',
      headers: {
        'Authorization': 'Basic ...',
        'Content-Type': 'application/x-www-form-urlencoded',
        'Content-Length': Buffer.byteLength(data)
      }
    };

    // Exchange the code for an access token
    const req = https.request(options, (res) => {
      console.log(`RESPONSE STATUS: ${res.statusCode}`);
      console.log(`RESPONSE HEADERS: ${JSON.stringify(res.headers)}`);

      let data = '';
    
      res.on('data', (chunk) => {
        data += chunk;
      });
    
      res.on('end', () => {
        console.log(`RESPONSE BODY: ${data}\n\n`);
      });
    });
    
    req.on('error', (error) => {
      console.error('Error:', error);
    });
    
    console.log(`REQUEST BODY: ${data}`)
    req.write(data);
    req.end();
  } else {
    res.write(JSON.stringify(params));
  }
  //params['state']
  res.end();
});

server.listen(port, hostname, () => {
  console.log(`Server running at http://${hostname}:${port}/`);
});
