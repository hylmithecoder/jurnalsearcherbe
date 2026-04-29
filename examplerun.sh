invoke_url='https://integrate.api.nvidia.com/v1/chat/completions'

authorization_header='Authorization: Bearer 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'
accept_header='Accept: application/json'
content_type_header='Content-Type: application/json'

data=$'{
  "model": "z-ai/glm4.7",
  "messages": [
    {
      "role": "user",
      "content": "hi"
    }
  ],
  "temperature": 0.7,
  "top_p": 1,
  "max_tokens": 16384,
  "seed": 42,
  "stream": false,
  "chat_template_kwargs": {
    "enable_thinking": true,
    "clear_thinking": false
  }
}'

response=$(curl --silent -i -w "\n%{http_code}" --request POST \
  --url "$invoke_url" \
  --header "$authorization_header" \
  --header "$accept_header" \
  --header "$content_type_header" \
  --data "$data"
)

echo "$response"