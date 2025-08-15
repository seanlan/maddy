#!/usr/bin/env python3
"""
测试 auth.pass_evm 邮箱登录验证
模拟以太坊签名验证过程
"""

import imaplib
import smtplib
from eth_account import Account
from eth_account.messages import encode_defunct
from email.mime.text import MIMEText
import logging

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


class BlockchainAuthTester:
    def __init__(self, imap_host='127.0.0.1', imap_port=993, 
                 smtp_host='127.0.0.1', smtp_port=587,
                 domain='fetm.top'):
        self.imap_host = imap_host
        self.imap_port = imap_port
        self.smtp_host = smtp_host
        self.smtp_port = smtp_port
        self.domain = domain
        
        # 创建一个测试用的以太坊账户
        self.account = Account.create()
        self.private_key = self.account.key
        self.address = self.account.address
        logger.info(f"Created test account: {self.address}")
        
    def create_signature(self, message):
        """
        创建以太坊签名
        按照 Go 代码中的格式：\x19Ethereum Signed Message:\n{长度}{消息}
        """
        # 构造符合以太坊标准的消息格式
        eth_message = encode_defunct(text=message)
        
        # 使用私钥签名
        signature = Account.sign_message(eth_message, self.private_key)
        
        # 返回十六进制格式的签名
        return signature.signature.hex()
    
    def create_username(self):
        """
        创建用户名：公钥@域名格式
        注意：这里用地址作为公钥部分（实际应用中可能需要真正的公钥）
        """
        return f"{self.address}@{self.domain}"
    
    def test_imap_login(self):
        """
        测试 IMAP 登录
        """
        username = self.create_username()
        
        # 创建签名：对地址（转小写）进行签名
        message = self.address.lower()
        signature = self.create_signature(message)
        
        logger.info(f"Testing IMAP login...")
        logger.info(f"Username: {username}")
        logger.info(f"Message to sign: {message}")
        logger.info(f"Signature: {signature}")
        
        try:
            # 连接到 IMAP 服务器
            if self.imap_port == 993:
                mail = imaplib.IMAP4_SSL(self.imap_host, self.imap_port)
            else:
                mail = imaplib.IMAP4(self.imap_host, self.imap_port)
                
            # 使用区块链认证登录
            # 用户名是地址@域名，密码是签名
            result = mail.login(username, signature)
            logger.info(f"IMAP login successful: {result}")
            
            # 列出邮箱
            _, folders = mail.list()
            logger.info(f"Mailbox folders: {folders}")
            
            # 选择收件箱
            mail.select('INBOX')
            
            # 搜索邮件
            _, messages = mail.search(None, 'ALL')
            logger.info(f"Email count: {len(messages[0].split()) if messages[0] else 0}")
            
            mail.logout()
            return True
            
        except Exception as e:
            logger.error(f"IMAP login failed: {e}")
            return False
    
    def test_smtp_login(self):
        """
        测试 SMTP 登录并发送邮件
        """
        username = self.create_username()
        
        # 创建签名：对地址（转小写）进行签名
        message = self.address.lower()
        signature = self.create_signature(message)
        
        logger.info(f"Testing SMTP login...")
        logger.info(f"Username: {username}")
        logger.info(f"Message to sign: {message}")
        logger.info(f"Signature: {signature}")
        
        try:
            # 连接到 SMTP 服务器
            if self.smtp_port == 465:
                server = smtplib.SMTP_SSL(self.smtp_host, self.smtp_port)
            else:
                server = smtplib.SMTP(self.smtp_host, self.smtp_port)
                server.starttls()
            
            # 使用区块链认证登录
            server.login(username, signature)
            logger.info("SMTP login successful")
            
            # 测试1: 发送到外部地址 (预期会被策略阻止)
            external_recipient = "0x018b80D73C26C6882B80F134449D0D2D92d1B737@fetm.top"
            msg1 = MIMEText("Test email to external address from blockchain auth", 'plain', 'utf-8')
            msg1['Subject'] = 'Blockchain Auth Test - External'
            msg1['From'] = username
            msg1['To'] = external_recipient
            
            logger.info(f"Attempting to send to external address: {external_recipient}")
            try:
                server.send_message(msg1)
                logger.info(f"External email sent successfully to {external_recipient}")
            except Exception as send_error:
                if "Non-local sender domain" in str(send_error):
                    logger.warning(f"External email blocked by server policy (expected): {send_error}")
                else:
                    logger.error(f"Unexpected external email error: {send_error}")
            
            # 测试2: 发送给自己 (应该成功，因为用户刚刚创建)
            self_recipient = username
            msg2 = MIMEText("Test email to self from blockchain auth system", 'plain', 'utf-8')
            msg2['Subject'] = 'Blockchain Auth Test - Self Email'
            msg2['From'] = username
            msg2['To'] = self_recipient
            
            logger.info(f"Attempting to send to self: {self_recipient}")
            try:
                server.send_message(msg2)
                logger.info(f"Self email sent successfully to {self_recipient}")
            except Exception as send_error:
                logger.error(f"Self email sending failed: {send_error}")
            
            # 测试3: 发送到postmaster (本地管理员)
            postmaster_recipient = f"postmaster@{self.domain}"
            msg3 = MIMEText("Test email to postmaster from blockchain auth", 'plain', 'utf-8')
            msg3['Subject'] = 'Blockchain Auth Test - Postmaster'
            msg3['From'] = username
            msg3['To'] = postmaster_recipient
            
            logger.info(f"Attempting to send to postmaster: {postmaster_recipient}")
            try:
                server.send_message(msg3)
                logger.info(f"Postmaster email sent successfully to {postmaster_recipient}")
            except Exception as send_error:
                if "User does not exist" in str(send_error):
                    logger.warning(f"Postmaster user not found (expected): {send_error}")
                else:
                    logger.error(f"Postmaster email sending failed: {send_error}")
                
            server.quit()
            return True
            
        except Exception as e:
            logger.error(f"SMTP login failed: {e}")
            return False
    
    def verify_signature_locally(self):
        """
        本地验证签名算法是否正确
        模拟 Go 代码中的 verifySignature 函数
        """
        logger.info("Testing local signature verification...")
        
        message = self.address.lower()
        signature = self.create_signature(message)
        
        try:
            # 使用 eth_account 验证签名
            eth_message = encode_defunct(text=message)
            
            # 处理签名格式 - 确保是正确的字节格式
            if signature.startswith('0x'):
                sig_bytes = bytes.fromhex(signature[2:])
            else:
                sig_bytes = bytes.fromhex(signature)
                
            recovered_address = Account.recover_message(eth_message, signature=sig_bytes)
            
            is_valid = recovered_address.lower() == self.address.lower()
            logger.info(f"Original address: {self.address}")
            logger.info(f"Recovered address: {recovered_address}")
            logger.info(f"Signature valid: {is_valid}")
            
            return is_valid
            
        except Exception as e:
            logger.error(f"Local signature verification failed: {e}")
            return False
    
    def verify_email_received(self):
        """
        验证邮件是否被接收
        """
        username = self.create_username()
        message = self.address.lower()
        signature = self.create_signature(message)
        
        try:
            # 重新连接IMAP查看邮件
            if self.imap_port == 993:
                mail = imaplib.IMAP4_SSL(self.imap_host, self.imap_port)
            else:
                mail = imaplib.IMAP4(self.imap_host, self.imap_port)
                
            # 登录
            mail.login(username, signature)
            
            # 选择收件箱
            mail.select('INBOX')
            
            # 搜索邮件
            _, messages = mail.search(None, 'ALL')
            email_count = len(messages[0].split()) if messages[0] else 0
            
            logger.info(f"Found {email_count} emails in inbox")
            
            if email_count > 0:
                # 获取最新邮件
                latest_msg_id = messages[0].split()[-1]
                _, msg_data = mail.fetch(latest_msg_id, '(RFC822)')
                
                # 简单显示邮件信息
                import email
                msg = email.message_from_bytes(msg_data[0][1])
                logger.info(f"Latest email subject: {msg.get('Subject')}")
                logger.info(f"Latest email from: {msg.get('From')}")
            
            mail.logout()
            return email_count > 0
            
        except Exception as e:
            logger.error(f"Email verification failed: {e}")
            return False

    def run_all_tests(self):
        """
        运行所有测试
        """
        logger.info("="*50)
        logger.info("Starting blockchain authentication tests")
        logger.info("="*50)
        
        # 首先验证签名算法
        if not self.verify_signature_locally():
            logger.error("Local signature verification failed, stopping tests")
            return False
        
        logger.info("-"*30)
        
        # 测试 IMAP 登录
        imap_success = self.test_imap_login()
        
        logger.info("-"*30)
        
        # 测试 SMTP 登录
        smtp_success = self.test_smtp_login()
        
        logger.info("-"*30)
        
        # 验证邮件接收
        logger.info("Verifying email delivery...")
        email_received = self.verify_email_received()
        
        logger.info("="*50)
        logger.info(f"Test Results:")
        logger.info(f"IMAP Login: {'✓' if imap_success else '✗'}")
        logger.info(f"SMTP Login: {'✓' if smtp_success else '✗'}")
        logger.info(f"Email Delivery: {'✓' if email_received else '✗'}")
        logger.info("="*50)
        
        return imap_success and smtp_success


def main():
    """
    主函数
    """
    # 创建测试实例
    # 注意：根据实际配置调整主机和端口
    tester = BlockchainAuthTester(
        imap_host='127.0.0.1',
        imap_port=993,  # SSL端口
        smtp_host='127.0.0.1', 
        smtp_port=587,  # submission 端口
        domain='fetm.top'
    )
    
    # 运行所有测试
    success = tester.run_all_tests()
    
    if success:
        logger.info("All tests passed!")
        return 0
    else:
        logger.error("Some tests failed!")
        return 1


if __name__ == '__main__':
    exit(main())