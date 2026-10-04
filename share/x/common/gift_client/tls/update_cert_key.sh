# 脚本概述：
  # 1. 重新生成CA证书，客户端与服务器的证书和秘钥。
# 何时需要执行此脚本：
  # 1. 任何一方（客户端、服务器、CA）证书泄露。
  # 2. 任何一方证书到期。
# 注意事项：
  # 1. -subj 中的/CN为域名，只有服务器对客户端做校验，而客户端不对服务器做校验的时候，才可以不关心。
  # 2. 如果需要客户端不对服务器做校验，请在客户端配置中设置 InsecureSkipVerify 参数为true。https库举例如下：
        #  	  tr := &http.Transport{
	      #	        TLSClientConfig: &tls.Config{
	      #		        RootCAs:            pool,
	      #		        Certificates:       []tls.Certificate{cliCrt},
	      #		        InsecureSkipVerify: true,
	      #	     },
	      #     }
	# 3. 由于只对证书的CA签名做验证，所以此处其实即使服务器和客户端证书互换或者相同也不会异常（设置InsecureSkipVerify时）
	      # 3_1. 此处生成两份只是为了易于区分。
# 拓展事项：
  # 1. 为什么要设置为true？
        # 1_1. 需求表达了GiftServer需要对GmTools和GameShard的身份验证，而不需要逆向认证。
        # 1_2. GiftServer的域名和IP可能会变动，而且证书在每个GameShard和GmTools下均有一份,

# --------------------------------------------CA秘钥与证书--------------------------------------------

# openssl 生成2048bit长度的RSA非对称CA秘钥。
openssl genrsa -out ca.key 2048
# openssl 生成x509格式的CA自签名证书（域名随意）。
openssl req -x509 -new -nodes -key ca.key -days 36500 -out ca.crt -subj "/CN=Don not care"

# ------------------------------------------服务器秘钥与证书-------------------------------------------

# openssl 生成2048bit长度的RSA非对称服务器秘钥。
openssl genrsa -out server.key 2048
# openssl 生成包含服务器公钥和域名的证书申请CSR文件。
openssl req -new -key server.key -out server.csr -subj "/CN=Don not care"
# openssl CA签署CSR文件生成服务器证书。
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 3650

# ------------------------------------------客户端秘钥与证书-------------------------------------------
# openssl 生成2048bit长度的RSA非对称客户端秘钥。
openssl genrsa -out client.key 2048
# openssl 生成包含客户端公钥和域名的证书申请CSR文件。
openssl req -new -key client.key -out client.csr -subj "/CN=Don not care"
# openssl CA签署CSR文件生成客户端证书。
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 365