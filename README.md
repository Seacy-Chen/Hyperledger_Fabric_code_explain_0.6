# Hyperledger_Fabric_code_explain_0.6

## Hyperledger Fabric 0.6版本代码详解

### 在fabric下有下面这些目录

LICENSE(apache LICENSE) 

Apache Licence是著名的非盈利开源组织Apache采用的协议。该协议和BSD类似，同样鼓励代码共享和尊重原作者的著作权，同样允许代码修改，再发布（作为开 源或商业软件）。需要满足的条件也和BSD类似：
* 需要给代码的用户一份Apache Licence
* 如果你修改了代码，需要在被修改的文件中说明。
* 在延伸的代码中（修改和有源代码衍生的代码中）需要带有原来代码中的协议，商标，专利声明和其他原来作者规定需要包含的说明。
* 如果再发布的产品中包含一个Notice文件，则在Notice文件中需要带有Apache Licence。你可以在Notice中增加自己的许可，但不可以表现为对Apache Licence构成更改。

TravisCI_Readme.md 

TravisCI是目前新兴的开源持续集成构建项目，它与jenkins，GO的很明显的特别在于采用yaml格式，简洁清新独树一帜。目前大多数的github项目都已经移入到TravisCI的构建队列中，据说Travis CI每天运行超过4000次完整构建。

core    

events	

gotools     

metadata	 

proposals  

scripts	     

tools

Makefile   

bddtests	       

devenv  

examples  

images      

mkdocs.yml  

protos     

sdk		     

vendor

consensus	       

docs   

flogging  

membersrvc

末端用户或者组织的身份发行与管理
作用：
 * 发行登记证书给各个末端使用者或者组织
 * 发行交易证书到关联的各个末端使用者
 * 发行TSL证书确保在HyperLedger Fabric之间通信
 * 发行链特别的Key

peer	 

pub	    

settings.gradle

目前只知道gradle是配置java链上代码的工具
