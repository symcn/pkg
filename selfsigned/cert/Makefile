CERT_PATH ?= .
SERVER_NAME ?= sym-webhook
SERVER_NAMESPACE ?= default
SUB_KEY_FILE_NAME ?= tls.key
SUB_CSR_FILE_NAME ?= tls.csr
SUB_CRT_FILE_NAME ?= tls.crt

build-cert:
	# build root cert
	openssl genrsa -out ${CERT_PATH}/ca.key 2048
	openssl req -new -x509 -key ${CERT_PATH}/ca.key -subj "/C=CN/ST=BJ/O=symcn, Inc./CN=*.${SERVER_NAMESPACE}.svc" -out ${CERT_PATH}/ca.crt -days 3650

	# build ${SERVER_NAME}.${SERVER_NAMESPACE}.svc cert use root cert
	openssl genrsa -out ${CERT_PATH}/${SUB_KEY_FILE_NAME} 2048
	openssl req -new -sha256 -key ${CERT_PATH}/${SUB_KEY_FILE_NAME} -subj "/C=CN/ST=BJ/O=symcn, Inc./CN=${SERVER_NAME}.${SERVER_NAMESPACE}.svc" -out ${CERT_PATH}/${SUB_CSR_FILE_NAME}
	openssl x509 -req -in ${CERT_PATH}/${SUB_CSR_FILE_NAME} -CA ${CERT_PATH}/ca.crt -CAkey ${CERT_PATH}/ca.key -CAcreateserial -out ${CERT_PATH}/${SUB_CRT_FILE_NAME} -days 3650 -sha256
