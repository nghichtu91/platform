### 敏感词文件

* 逻辑：将用户上传的敏感词文件上传到oss，同时更新etcd
  * Etcd: 
    * key: [DevopsEtcdRoot]/[大区号]/chat_illegal_words_hash_path   
    * Value: 文件的md5值
  * Oss: 
    * taiyouxi-chat-data/illegal_words_11_common_illegal.csv



